package nomad

import (
	"time"

	"github.com/hashicorp/nomad/api"
)

func CreateJob(jobName string, value string, distinctHosts bool, stagger time.Duration) api.Job {
	region := "eu"
	jobType := "service"

	job := api.Job{
		Name:        &jobName,
		Region:      &region,
		Datacenters: []string{"eu-west-1"},
		Type:        &jobType,
	}

	createUpdateStrategy(stagger)
	createTaskGroup(value, distinctHosts)

	return job
}

func createUpdateStrategy(stagger time.Duration) api.UpdateStrategy {
	healthyTime := time.Second * 30
	healthyDeadline := time.Minute * 2
	maxParallel := 1
	autorevert := true

	updateStrategy := api.UpdateStrategy{
		Stagger:         &stagger, //Java 150s Go 60s
		MinHealthyTime:  &healthyTime,
		HealthyDeadline: &healthyDeadline,
		MaxParallel:     &maxParallel,
		AutoRevert:      &autorevert,
	}

	return updateStrategy
}

func createTaskGroup(value string, distinctHosts bool) {
	for tasks {
	}

	createConstraint(value, distinctHosts)
}

func createConstraint(value string, distinctHosts bool) api.Constraint {
	// bool for distinct hosts the if statement so if bool set operand to "distinct_hosts"
	if !distinctHosts {
		return api.Constraint{
			LTarget: "${node.class}",
			RTarget: value,
		}
	} else {
		return api.Constraint{
			LTarget: "${node.class}",
			RTarget: value,            // publishing, web, web-mount or publishing-mount
			Operand: "distinct_hosts", // is set for Java
		}
	}
}
