package nomad

import (
	"time"

	"github.com/hashicorp/nomad/api"
)

func CreateJob(jobName string, value string, distinctHosts bool, stagger time.Duration) {
	job := api.Job {
		Name: jobName,
		Region: "eu",
		Datacenters: ["eu-west-1"],
		Type: "service",
	}

	createUpdateStrategy(stagger)
	createTaskGroup(vaue, distinctHosts)
}

func createUpdateStrategy(stagger time.Duration) {
	updateStrategy := api.UpdateStrategy {
		Stagger: , //Java 150s Go 60s
		MinHealthyTime: "30s",
		HealthyDeadline: "2m",
		MaxParallel: 1,
		AutoRevert: true,
	}

	return updateStrategy
}

func createTaskGroup() {
	for tasks {}

	createConstraint()
}

func createConstraint(value string, distinctHosts bool) {
	constraint := api.Constraint {
		LTarget: ,
		RTarget: value, // publishing, web, web-mount or publishing-mount
		Operand: distinctHosts, // true for Java
	}

	return constraint
}