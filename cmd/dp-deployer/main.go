package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ONSdigital/dp-deployer/engine"
	"github.com/ONSdigital/dp-deployer/handler/deployment"
	"github.com/ONSdigital/dp-deployer/handler/secret"
	"github.com/ONSdigital/go-ns/log"
	"github.com/namsral/flag"
)

var (
	consumerQueue    = flag.String("consumer-queue", "", "sqs consumer queue name")
	consumerQueueURL = flag.String("consumer-queue-url", "", "sqs queue url")
	deploymentRoot   = flag.String("deployment-root", "", "root deployment directory")
	nomadEndpoint    = flag.String("nomad-endpoint", "http://localhost:4646", "nomad client endpoint")
	nomadToken       = flag.String("nomad-token", "", "nomad acl token")
	nomadCACert      = flag.String("nomad-ca-cert", "", "nomad CA cert file")
	privateKey       = flag.String("private-key", "", "private key used to decrypt secrets")
	producerQueue    = flag.String("producer-queue", "", "sqs producer queue name")
	region           = flag.String("aws-default-region", "", "sqs queue region")
	verificationKey  = flag.String("verification-key", "", "public key for verifying queue messages")
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
		NomadToken:     *nomadToken,
		NomadCACert:    *nomadCACert,
		Region:         *region,
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
