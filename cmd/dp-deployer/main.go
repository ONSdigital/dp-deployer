package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ONSdigital/dp-deployer/engine"
	"github.com/ONSdigital/dp-deployer/handler/deployment"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/ONSdigital/dp-deployer/handler/secret"
	"github.com/ONSdigital/go-ns/log"
	"github.com/namsral/flag"
)

var (
	consumerQueue      = flag.String("consumer-queue", "", "sqs consumer queue name")
	consumerQueueURL   = flag.String("consumer-queue-url", "", "sqs queue url")
	deploymentRoot     = flag.String("deployment-root", "", "root deployment directory")
	nomadEndpoint      = flag.String("nomad-endpoint", "http://localhost:4646", "nomad client endpoint")
	nomadTLSSkipVerify = flag.Bool("nomad-tls-skip-verify", false, "skip tls verification of nomad cert")
	nomadToken         = flag.String("nomad-token", "", "nomad acl token")
	nomadCACert        = flag.String("nomad-ca-cert", "", "nomad CA cert file")
	privateKey         = flag.String("private-key", "", "private key used to decrypt secrets")
	producerQueue      = flag.String("producer-queue", "", "sqs producer queue name")
	region             = flag.String("aws-default-region", "", "sqs queue region")
	verificationKey    = flag.String("verification-key", "", "public key for verifying queue messages")
	healthcheckInterval = flag.String("healthcheck-interval", "10s", "time between calling healthcheck endpoints for check subsystems")
	healthcheckCriticalTimeout = flag.String("healthcheck-critical-timeout", "60s", "time taken for the health changes from warning state to critical due to subsystem check failures")
)

var (
	// BuildTime represents the time in which the service was built
	BuiltTime string
	// GitCommit represents the commit (SHA-1) hash of the service that is running
	GitCommit string
	// Version represents the version of the service that is running
	Version string
)

var wg sync.WaitGroup

func main() {
	log.Namespace = "dp-deployer"
	flag.Parse()

	h, err := initHandlers()
	if err != nil {
		log.Error(err, nil)
		os.Exit(1)
	}

	ec := &engine.Config{
		ConsumerQueue:    *consumerQueue,
		ConsumerQueueURL: *consumerQueueURL,
		ProducerQueue:    *producerQueue,
		Region:           *region,
		VerificationKey:  *verificationKey,
	}
	e, err := engine.New(ec, h)
	if err != nil {
		log.Error(err, nil)
		os.Exit(1)
	}

	hcc := &healthcheck.Config{
		HealthcheckInterval: *healthcheckInterval,
		HealthcheckCriticalTimeout: *healthcheckCriticalTimeout,
	}

	hc, err := service.List.GetHealthCheck(cfg, BuildTime, GitCommit, Version)
	if err != nill {
		log.Event(ctx, "failed tp create service version information", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	r := mux.NewRouter()
	r.HandlerFunc("/health", hc.Handler)

	// Start healthcheck
	hc.Start(ctx)

	// Create and start http server for healthcheck
	httpServer := server.New(cfg.BindAddr, r)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			log.Event(ctx, "failed to start healthcheck HTTP server", log.FATAL, log.Data{"config": hcc}, log.Error(err))
			os.Exit(2)
		}
	}()

	sigC := make(chan os.Signal)
	signal.Notify(sigC, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())

	wg.Add(1)
	go func() {
		defer wg.Done()
		e.Start(ctx)
	}()

	sig := <-sigC
	log.Info("received exit signal", log.Data{"signal": sig})
	cancel()
	wg.Wait()
}

func initHandlers() (map[string]engine.HandlerFunc, error) {
	dc := &deployment.Config{
		DeploymentRoot:     *deploymentRoot,
		NomadEndpoint:      *nomadEndpoint,
		NomadTLSSkipVerify: *nomadTLSSkipVerify,
		NomadToken:         *nomadToken,
		NomadCACert:        *nomadCACert,
		Region:             *region,
	}
	d, err := deployment.New(dc)
	if err != nil {
		return nil, err
	}

	sc := &secret.Config{
		PrivateKey: *privateKey,
		Region:     *region,
	}
	s, err := secret.New(sc)
	if err != nil {
		return nil, err
	}

	return map[string]engine.HandlerFunc{
		"deployment": d.Handler,
		"secret":     s.Handler,
	}, nil
}
