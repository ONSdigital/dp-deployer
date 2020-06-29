package nomad

import (
	"time"

	"github.com/hashicorp/nomad/api"
)

const webSuffix = "-web"
const publishingSuffix = "-publishing"

func CreateJob(name string, value string, distinctHosts bool,
	args []string, volumes []string, userns_mode string, isJava bool,
	isPublishing bool, isWeb bool) api.Job {
	region := "eu"
	jobType := "service"

	job := api.Job{
		Name:        &name,
		Region:      &region,
		Datacenters: []string{"eu-west-1"},
		Type:        &jobType,
	}

	createUpdateStrategy(isJava)
	createTaskGroup(name, value, distinctHosts, args, volumes, userns_mode)

	return job
}

func createUpdateStrategy(isJava bool) api.UpdateStrategy {
	healthyTime := time.Second * 30
	healthyDeadline := time.Minute * 2
	maxParallel := 1
	autorevert := true
	stagger := time.Second * 60

	if isJava {
		stagger = time.Second * 150
	}

	updateStrategy := api.UpdateStrategy{
		Stagger:         &stagger,
		MinHealthyTime:  &healthyTime,
		HealthyDeadline: &healthyDeadline,
		MaxParallel:     &maxParallel,
		AutoRevert:      &autorevert,
	}

	return updateStrategy
}

func createTaskGroup(name string, value string, distinctHosts bool, args []string,
	volumes []string, userns_mode string) {
	// validation for goup and pub / web here first
	// group name var needed too
	for tasks {
		createTask(name, args, volumes, userns_mode)
	}
	if distinctHosts {
		createConstraint("", distinctHosts)
	}
	createConstraint(value, false)
}

func createConstraint(value string, distinctHosts bool) api.Constraint {
	if !distinctHosts {
		return api.Constraint{
			LTarget: "${node.class}",
			RTarget: value,
		}
	}

	return api.Constraint{
		Operand: "distinct_hosts", // is set for Java
	}
}
