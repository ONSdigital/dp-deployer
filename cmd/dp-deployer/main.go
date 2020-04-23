package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ONSdigital/dp-deployer/config"
	"github.com/ONSdigital/dp-deployer/engine"
	"github.com/ONSdigital/dp-deployer/handler/deployment"
	"github.com/ONSdigital/dp-deployer/handler/secret"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	s3client "github.com/ONSdigital/dp-s3"
	vault "github.com/ONSdigital/dp-vault"
	"github.com/ONSdigital/go-ns/server"
	"github.com/ONSdigital/log.go/log"
	"github.com/gorilla/mux"
)

var (
	// BuildTime represents the time in which the service was built
	BuildTime string
	// GitCommit represents the commit (SHA-1) hash of the service that is running
	GitCommit string
	// Version represents the version of the service that is running
	Version string
)

var wg sync.WaitGroup

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	log.Namespace = "dp-deployer"

	cfg, err := config.Get()
	if err != nil {
		log.Event(ctx, "Failed to initialise config", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	log.Event(ctx, "config on startup", log.INFO, log.Data{"config": cfg})

	// Create vault client
	var vc *vault.Client
	vc, err = vault.CreateClient(cfg.VaultToken, cfg.VaultAddr, 3)
	if err != nil {
		log.Event(ctx, "error creating vault client", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	// Create S3 secrets client
	var s3sc *s3client.S3
	s3sc, err = s3client.NewClient(cfg.AWSRegion, cfg.SecretsBucketName, false)
	if err != nil {
		log.Event(ctx, "error creating S3 secrets client", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	// Create S3 deployments client
	var s3dc *s3client.S3
	s3dc, err = s3client.NewClient(cfg.AWSRegion, cfg.DeploymentsBucketName, false)
	if err != nil {
		log.Event(ctx, "error creating S3 deployments client", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	h, err := initHandlers(ctx, cfg, vc)
	if err != nil {
		log.Event(ctx, "failed to initialise handlers", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	e, err := engine.New(cfg, h)
	if err != nil {
		log.Event(ctx, "failed to create engine", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	hc := startHealthChecks(ctx, cfg, vc, s3sc, s3dc)

	r := mux.NewRouter()
	r.HandleFunc("/health", hc.Handler)

	sigC := make(chan os.Signal)
	signal.Notify(sigC, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	wg.Add(1)
	go func() {
		defer wg.Done()
		e.Start(ctx)
	}()

	// Create and start http server for healthcheck
	httpServer := server.New(cfg.BindAddr, r)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			log.Event(ctx, "error starting http server", log.Error(err))
			cancel()
		}
	}()

	select {
	case sig := <-sigC:
		log.Event(ctx, "received exit signal", log.ERROR, log.Data{"signal": sig})
		cancel()
	case <-ctx.Done():
		log.Event(ctx, "context done", log.INFO)
	}
	wg.Wait()
}

func initHandlers(ctx context.Context, cfg *config.Configuration, vc *vault.Client) (map[string]engine.HandlerFunc, error) {
	d, err := deployment.New(ctx, cfg)
	if err != nil {
		return nil, err
	}

	s, err := secret.New(cfg, vc)
	if err != nil {
		return nil, err
	}

	return map[string]engine.HandlerFunc{
		"deployment": d.Handler,
		"secret":     s.Handler,
	}, nil
}

func startHealthChecks(ctx context.Context, cfg *config.Configuration, vaultChecker *vault.Client, s3sChecker *s3client.S3, s3dChecker *s3client.S3) *healthcheck.HealthCheck {
	hasErrors := false

	// Create healthcheck object with versionInfo
	versionInfo, err := healthcheck.NewVersionInfo(BuildTime, GitCommit, Version)
	if err != nil {
		log.Event(ctx, "failed to create service version information", log.FATAL, log.Error(err))
		os.Exit(1)
	}
	hc := healthcheck.New(versionInfo, cfg.HealthcheckCriticalTimeout, cfg.HealthcheckInterval)

	if err := hc.AddCheck("Vault", vaultChecker.Checker); err != nil {
		hasErrors = true
		log.Event(ctx, "error adding check for vault", log.ERROR, log.Error(err))
	}

	if err := hc.AddCheck("S3 secret", s3sChecker.Checker); err != nil {
		hasErrors = true
		log.Event(ctx, "error adding check for S3 secrets", log.ERROR, log.Error(err))
	}

	if err := hc.AddCheck("S3 deployment", s3dChecker.Checker); err != nil {
		hasErrors = true
		log.Event(ctx, "error adding check for S3 deployments", log.ERROR, log.Error(err))
	}

	if hasErrors {
		os.Exit(1)
	}

	// Start healthcheck
	hc.Start(ctx)

	return &hc
}
