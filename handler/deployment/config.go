package deployment

import "time"

// Config represents the configuration for a deployment.
type Config struct {
	// DeploymentRoot is the path to root of deployments.
	DeploymentRoot string
	// NomadEndpoint is the Nomad client endpoint.
	NomadEndpoint string
	// Region is the region in which the queues reside.
	Region string
	// TImeout is the timeout configuration for the deployments.
	Timeout *TimeoutConfig
}

// TimeoutConfig represents the configuration for deployment timeouts.
type TimeoutConfig struct {
	// Allocation is the max time to wait for all allocations to complete.
	Allocation time.Duration
	// Evaluation is the max time to wait for an Evaluation to complete.
	Evaluation time.Duration
}
