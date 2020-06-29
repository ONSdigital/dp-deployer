package nomad

import (
	"time"

	"github.com/hashicorp/nomad/api"

	"github.com/ONSdigital/dp-deployer/message"
)

func CreateJob(name string, isJava bool, publishing *message.Groups, web *message.Groups) api.Job {
	region := "eu"
	jobType := "service"

	job := api.Job{
		Name:        &name,
		Region:      &region,
		Datacenters: []string{"eu-west-1"},
		Type:        &jobType,
	}

	createUpdateStrategy(isJava)

	if publishing != nil {
		createTaskGroup(name, "publishing", publishing)
	}
	if web != nil {
		createTaskGroup(name, "web", web)
	}

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

func createTaskGroup(name string, groupName string, details *message.Groups) {
	// validation for goup and pub / web here first
	// group name var needed too
	for tasks {
		createTask(name+"-"+groupName, details)
	}

	if details.DistinctHosts {
		createConstraint("", details.DistinctHosts)
	}
	createConstraint(groupName, false)
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
