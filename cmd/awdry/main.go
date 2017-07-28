package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ONSdigital/dp-ci/awdry/engine"
	"github.com/ONSdigital/dp-ci/awdry/handler/deployment"
	"github.com/ONSdigital/dp-ci/awdry/handler/secret"
	"github.com/ONSdigital/go-ns/log"
	"github.com/namsral/flag"
)

var (
	consumerQueue    = flag.String("consumer-queue", "", "sqs consumer queue name")
	consumerQueueURL = flag.String("consumer-queue-url", "", "sqs queue url")
	deploymentRoot   = flag.String("deployment-root", "", "root deployment directory")
	nomadEndpoint    = flag.String("nomad-endpoint", "http://localhost:4646", "nomad client endpoint")
	privateKeyPath   = flag.String("private-key-path", "", "path to private key")
	producerQueue    = flag.String("producer-queue", "", "sqs producer queue name")
	region           = flag.String("aws-default-region", "", "sqs queue region")
)

var wg sync.WaitGroup

func main() {
	log.Namespace = "awdry"
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
	}
	e, err := engine.New(ec, h)
	if err != nil {
		log.Error(err, nil)
		os.Exit(1)
	}

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
		DeploymentRoot: *deploymentRoot,
		NomadEndpoint:  *nomadEndpoint,
		Region:         *region,
	}
	d, err := deployment.New(dc)
	if err != nil {
		return nil, err
	}

	sc := &secret.Config{
		PrivateKeyPath: *privateKeyPath,
		Region:         *region,
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
