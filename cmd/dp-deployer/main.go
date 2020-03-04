package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ONSdigital/dp-deployer/engine"
	"github.com/ONSdigital/dp-deployer/handler/deployment"
	"github.com/ONSdigital/dp-deployer/handler/secret"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/ONSdigital/go-ns/server"
	"github.com/ONSdigital/log.go/log"
	"github.com/gorilla/mux"
	"github.com/namsral/flag"
)

var (
	consumerQueue              = flag.String("consumer-queue", "", "sqs consumer queue name")
	consumerQueueURL           = flag.String("consumer-queue-url", "", "sqs queue url")
	deploymentRoot             = flag.String("deployment-root", "", "root deployment directory")
	nomadEndpoint              = flag.String("nomad-endpoint", "http://localhost:4646", "nomad client endpoint")
	nomadTLSSkipVerify         = flag.Bool("nomad-tls-skip-verify", false, "skip tls verification of nomad cert")
	nomadToken                 = flag.String("nomad-token", "", "nomad acl token")
	nomadCACert                = flag.String("nomad-ca-cert", "", "nomad CA cert file")
	privateKey                 = flag.String("private-key", "", "private key used to decrypt secrets")
	producerQueue              = flag.String("producer-queue", "", "sqs producer queue name")
	region                     = flag.String("aws-default-region", "", "sqs queue region")
	verificationKey            = flag.String("verification-key", "", "public key for verifying queue messages")
	healthcheckInterval        = flag.String("healthcheck-interval", "10s", "time between calling healthcheck endpoints for check subsystems")
	healthcheckCriticalTimeout = flag.String("healthcheck-critical-timeout", "60s", "time taken for the health changes from warning state to critical due to subsystem check failures")
	bindAddr                   = flag.String("bind-addr", ":24300", "The listen address to bind to")
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

type healthcheckConfig struct {
	IntervalStr                string
	CriticalTimeoutStr         string
	BindAddr                   string
	HealthcheckInterval        time.Duration
	HealthcheckCriticalTimeout time.Duration
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	log.Namespace = "dp-deployer"
	flag.Parse()

	h, err := initHandlers(ctx)
	if err != nil {
		log.Event(ctx, "failed to initialise handlers", log.FATAL, log.Error(err))
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
		log.Event(ctx, "failed to create engine", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	//TODO: remove this when config is not driven by flags
	healthInterval, err := time.ParseDuration(*healthcheckInterval)
	if err != nil {
		log.Event(ctx, "healthInterval parse failed", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	healthTimeout, err := time.ParseDuration(*healthcheckCriticalTimeout)
	if err != nil {
		log.Event(ctx, "healthTimeout parse failed", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	hcc := &healthcheckConfig{
		HealthcheckInterval:        healthInterval,
		HealthcheckCriticalTimeout: healthTimeout,
		BindAddr:                   *bindAddr,
	}

	// Create healthcheck object with versionInfo
	versionInfo, err := healthcheck.NewVersionInfo(BuildTime, GitCommit, Version)
	if err != nil {
		log.Event(ctx, "failed to create service version information", log.FATAL, log.Error(err))
		os.Exit(1)
	}
	hc := healthcheck.New(versionInfo, hcc.HealthcheckCriticalTimeout, hcc.HealthcheckInterval)

	r := mux.NewRouter()
	r.HandleFunc("/health", hc.Handler)

	// Start healthcheck
	hc.Start(ctx)

	sigC := make(chan os.Signal)
	signal.Notify(sigC, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	wg.Add(1)
	go func() {
		defer wg.Done()
		e.Start(ctx)
	}()

	// Create and start http server for healthcheck
	httpServer := server.New(hcc.BindAddr, r)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			log.Event(ctx, "error starting http server", log.Error(err), log.Data{"config": hcc})
			cancel()
		}
	}()

	select {
	case sig := <-sigC:
		log.Event(ctx, "received exit signal", log.INFO, log.Data{"signal": sig})
		cancel()
	case <-ctx.Done():
		log.Event(ctx, "context done", log.INFO)
	}
	wg.Wait()
}

func initHandlers(ctx context.Context) (map[string]engine.HandlerFunc, error) {
	dc := &deployment.Config{
		DeploymentRoot:     *deploymentRoot,
		NomadEndpoint:      *nomadEndpoint,
		NomadTLSSkipVerify: *nomadTLSSkipVerify,
		NomadToken:         *nomadToken,
		NomadCACert:        *nomadCACert,
		Region:             *region,
	}
	d, err := deployment.New(ctx, dc)
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
