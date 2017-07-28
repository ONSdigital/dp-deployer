package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ONSdigital/dp-ci/awdry/engine"
	"github.com/ONSdigital/dp-ci/awdry/handler/deployment"
	"github.com/ONSdigital/go-ns/log"
	"github.com/namsral/flag"
)

var (
	consumerQueue    = flag.String("consumer-queue", "", "sqs consumer queue name")
	consumerQueueURL = flag.String("consumer-queue-url", "", "sqs queue url")
	deploymentRoot   = flag.String("deployment-root", "", "root deployment directory")
	nomadEndpoint    = flag.String("nomad-endpoint", "http://localhost:4646", "nomad client endpoint")
	producerQueue    = flag.String("producer-queue", "", "sqs producer queue name")
	region           = flag.String("aws-default-region", "", "sqs queue region")
)

var wg sync.WaitGroup

func main() {
	log.Namespace = "awdry"
	flag.Parse()

	dc := &deployment.Config{
		DeploymentRoot: *deploymentRoot,
		NomadEndpoint:  *nomadEndpoint,
		Region:         *region,
	}
	ec := &engine.Config{
		ConsumerQueue:    *consumerQueue,
		ConsumerQueueURL: *consumerQueueURL,
		ProducerQueue:    *producerQueue,
		Region:           *region,
	}

	h, err := deployment.New(dc)
	if err != nil {
		log.Error(err, log.Data{"configuration": dc})
		os.Exit(1)
	}
	e, err := engine.New(ec, map[string]engine.HandlerFunc{"deployment": h.Handler})
	if err != nil {
		log.Error(err, log.Data{"configuration": ec})
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
