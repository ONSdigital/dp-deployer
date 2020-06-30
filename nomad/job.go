package nomad

import (
	"context"
	"errors"
	"time"

	"github.com/hashicorp/nomad/api"

	"github.com/ONSdigital/dp-deployer/message"
	"github.com/ONSdigital/log.go/log"
)

// CreateJob Creates the Nomad job structure for an application deployment
func CreateJob(ctx context.Context, name string, jobStruct *message.MessageSQS, publishing *message.Groups,
	web *message.Groups, healthcheck *message.Healthcheck) api.Job {
	region := "eu"
	jobType := "service"

	job := api.Job{
		Name:        &name,
		Region:      &region,
		Datacenters: []string{"eu-west-1"},
		Type:        &jobType,
	}

	createUpdateStrategy(jobStruct.Java)

	if publishing != nil {
		createTaskGroup(ctx, name, "publishing", publishing, healthcheck, jobStruct.Revision)
	}
	if web != nil {
		createTaskGroup(ctx, name, "web", web, healthcheck, jobStruct.Revision)
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

func createTaskGroup(ctx context.Context, name string, groupName string, details *message.Groups, healthcheck *message.Healthcheck,
	revision string) (*api.TaskGroup, error) {

	if groupName != "web" && groupName != "publishing" {
		err := errors.New("Not a valid group name")
		log.Event(ctx, err.Error(), log.ERROR, log.Data{"group_name": groupName})
		return nil, err
	}

	taskGroup := api.TaskGroup{
		Name:  &groupName,
		Count: &details.TaskCount,
	}
	createTask(name+"-"+groupName, details, revision)

	if details.DistinctHosts {
		createConstraint("", details.DistinctHosts)
	}
	createConstraint(groupName, false)

	createService(name, groupName, healthcheck)

	return &taskGroup, nil
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
