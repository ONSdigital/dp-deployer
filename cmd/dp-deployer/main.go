package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ONSdigital/dp-deployer/config"
	"github.com/ONSdigital/dp-deployer/engine"
	"github.com/ONSdigital/dp-deployer/handler/deployment"
	"github.com/ONSdigital/dp-deployer/handler/secret"
	"github.com/ONSdigital/dp-deployer/queue"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	nomad "github.com/ONSdigital/dp-nomad"
	s3client "github.com/ONSdigital/dp-s3"
	vault "github.com/ONSdigital/dp-vault"
	"github.com/ONSdigital/dp-net/http"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

var (
	// BuildTime represents the time in which the service was built
	BuildTime string
	// GitCommit represents the commit (SHA-1) hash of the service that is running
	GitCommit string
	// Version represents the version of the service that is running
	Version string
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	log.Namespace = "dp-deployer"

	cfg, err := config.Get()
	if err != nil {
		log.Fatal(ctx, "Failed to initialise config", err)
		os.Exit(1)
	}

	log.Info(ctx, "config on startup", log.Data{"config": cfg})

	// Create vault client
	var vc *vault.Client
	vc, err = vault.CreateClient(cfg.VaultToken, cfg.VaultAddr, 3)
	if err != nil {
		log.Fatal(ctx, "error creating vault client", err)
		os.Exit(1)
	}

	// Create S3 secrets client
	var secretsClient *s3client.S3
	secretsClient, err = s3client.NewClient(cfg.AWSRegion, cfg.SecretsBucketName)
	if err != nil {
		log.Fatal(ctx, "error creating S3 secrets client", err)
		os.Exit(1)
	}

	// Create S3 deployments client
	var deploymentsClient *s3client.S3
	deploymentsClient, err = s3client.NewClient(cfg.AWSRegion, cfg.DeploymentsBucketName)
	if err != nil {
		log.Fatal(ctx, "error creating S3 deployments client", err)
		os.Exit(1)
	}

	// create Nomad client
	var nomadClient *nomad.Client
	nomadClient, err = nomad.NewClient(cfg.NomadEndpoint, cfg.NomadCACert, cfg.NomadTLSSkipVerify)
	if err != nil {
		log.Fatal(ctx, "error creating nomad client", err)
		os.Exit(1)
	}

	// TODO: remove once new queue implemented fully
	oldHandler, err := initHandlersOld(cfg, vc, deploymentsClient, secretsClient, nomadClient)
	if err != nil {
		log.Fatal(ctx, "failed to initialise handlers", err)
		os.Exit(1)
	}

	e, err := engine.New(cfg, oldHandler)
	if err != nil {
		log.Fatal(ctx, "failed to create engine", err)
		os.Exit(1)
	}

	h, err := initHandlers(cfg, vc, deploymentsClient, secretsClient, nomadClient)
	if err != nil {
		log.Fatal(ctx, "failed to initialise handlers", err)
		os.Exit(1)
	}

	q, err := queue.New(cfg, h)
	if err != nil {
		log.Fatal(ctx, "failed to create engine", err)
		os.Exit(1)
	}

	hc, err := startHealthChecks(ctx, cfg, vc, secretsClient, deploymentsClient, nomadClient)
	if err != nil {
		log.Fatal(ctx, "failed to start healthchecks", err)
		os.Exit(1)
	}

	r := mux.NewRouter()
	r.HandleFunc("/health", hc.Handler)

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	// TODO: remove once new queue implemented fully
	go func() {
		e.Start(ctx)
	}()

	go func() {
		q.Start(ctx)
	}()

	// Create and start http server for healthcheck
	httpServer := http.NewServer(cfg.BindAddr, r)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			log.Error(ctx, "error starting http server", err)
			cancel()
		}
	}()

	select {
	case sig := <-sigC:
		log.Error(ctx, "received exit signal", errors.New("received exit signal"), log.Data{"signal": sig})
		cancel()
	case <-ctx.Done():
		log.Info(ctx, "context done")
	}

	log.Info(ctx, "shutdown with timeout:", log.Data{"Timeout": cfg.GracefulShutdownTimeout})
	shutdownContext, shutdownCtxCancel := context.WithTimeout(context.Background(), cfg.GracefulShutdownTimeout)

	go func() {

		// Shutdown HTTP server
		log.Info(shutdownContext, "closing http server")
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Error(shutdownContext, "failed to gracefully close http server", err)
		}
		log.Info(ctx, "http server gracefully closed ")

		// Stop healthcheck
		hc.Stop()

		e.Close()

		shutdownCtxCancel()
	}()

	<-shutdownContext.Done()
	if shutdownContext.Err() == context.DeadlineExceeded {
		log.Error(shutdownContext, "shutdown timeout", shutdownContext.Err())
		os.Exit(1)
	} else {
		log.Error(shutdownContext, "done shutdown gracefully", errors.New("done shutdown gracefully"), log.Data{"context": shutdownContext.Err()})
		os.Exit(0)
	}

}

// TODO: remove once new queue implemented fully
func initHandlersOld(cfg *config.Configuration, vc *vault.Client, deploymentsClient *s3client.S3, secretsClient *s3client.S3, nomadClient *nomad.Client) (map[string]engine.HandlerFunc, error) {
	d := deployment.New(cfg, deploymentsClient, nomadClient)

	s, err := secret.New(cfg, vc, secretsClient)
	if err != nil {
		return nil, err
	}

	return map[string]engine.HandlerFunc{
		"deployment": d.Handler,
		"secret":     s.Handler,
	}, nil
}

func initHandlers(cfg *config.Configuration, vc *vault.Client, deploymentsClient *s3client.S3, secretsClient *s3client.S3, nomadClient *nomad.Client) (queue.HandlerFunc, error) {
	d := deployment.New(cfg, deploymentsClient, nomadClient)

	return d.NewHandler, nil
}

func startHealthChecks(ctx context.Context, cfg *config.Configuration, vaultChecker *vault.Client, s3sChecker *s3client.S3, s3dChecker *s3client.S3, nomadClient *nomad.Client) (*healthcheck.HealthCheck, error) {

	// Create healthcheck object with versionInfo
	versionInfo, err := healthcheck.NewVersionInfo(BuildTime, GitCommit, Version)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create service version information")
	}
	hc := healthcheck.New(versionInfo, cfg.HealthcheckCriticalTimeout, cfg.HealthcheckInterval)

	if err := hc.AddCheck("Vault", vaultChecker.Checker); err != nil {
		return nil, errors.Wrap(err, "error adding check for vault")
	}

	if err := hc.AddCheck("S3 secret", s3sChecker.Checker); err != nil {
		return nil, errors.Wrap(err, "error adding check for S3 secrets")
	}

	if err := hc.AddCheck("S3 deployment", s3dChecker.Checker); err != nil {
		return nil, errors.Wrap(err, "error adding check for S3 deployments")
	}

	if err := hc.AddCheck("Nomad", nomadClient.Checker); err != nil {
		return nil, errors.Wrap(err, "error adding check for nomad")
	}

	// Start healthcheck
	hc.Start(ctx)

	return &hc, nil
}
