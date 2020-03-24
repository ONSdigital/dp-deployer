package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ONSdigital/dp-deployer/config"
	"github.com/ONSdigital/dp-deployer/engine"
	"github.com/ONSdigital/dp-deployer/handler/deployment"
	"github.com/ONSdigital/dp-deployer/handler/secret"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
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

	cfg, err := config.Get()
	if err != nil {
		log.Event(ctx, "Failed to initialise config", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	log.Event(ctx, "config on startup", log.INFO, log.Data{"config": cfg})

	h, err := initHandlers(ctx, cfg)
	if err != nil {
		log.Event(ctx, "failed to initialise handlers", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	e, err := engine.New(cfg, h)
	if err != nil {
		log.Event(ctx, "failed to create engine", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	// Create healthcheck object with versionInfo
	versionInfo, err := healthcheck.NewVersionInfo(BuildTime, GitCommit, Version)
	if err != nil {
		log.Event(ctx, "failed to create service version information", log.FATAL, log.Error(err))
		os.Exit(1)
	}
	hc := healthcheck.New(versionInfo, cfg.HealthcheckCriticalTimeout, cfg.HealthcheckInterval)

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

func initHandlers(ctx context.Context, cfg *config.Configuration) (map[string]engine.HandlerFunc, error) {
	d, err := deployment.New(ctx, cfg)
	if err != nil {
		return nil, err
	}

	s, err := secret.New(cfg)
	if err != nil {
		return nil, err
	}

	return map[string]engine.HandlerFunc{
		"deployment": d.Handler,
		"secret":     s.Handler,
	}, nil
}
