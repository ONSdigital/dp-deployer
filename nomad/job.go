package nomad

import (
	"context"
	"errors"
	"time"

	"github.com/hashicorp/nomad/api"

	"github.com/ONSdigital/dp-deployer/config"
	"github.com/ONSdigital/dp-deployer/message"
	"github.com/ONSdigital/log.go/log"
)

// CreateJob Creates the Nomad job structure for an application deployment
func CreateJob(ctx context.Context, cfg *config.Configuration, name string, jobStruct *message.MessageSQS) api.Job {
	region := "eu"
	jobType := "service"

	updateStrategy := createUpdateStrategy(jobStruct.Java)
	var taskGroups []*api.TaskGroup

	if jobStruct.Publishing != nil {
		taskGroup1, _ := createTaskGroup(ctx, cfg, name, "publishing", jobStruct.Publishing, jobStruct.Healthcheck, jobStruct.Revision)
		taskGroups = append(taskGroups, taskGroup1)
	}
	if jobStruct.Web != nil {
		taskGroup2, _ := createTaskGroup(ctx, cfg, name, "web", jobStruct.Web, jobStruct.Healthcheck, jobStruct.Revision)
		taskGroups = append(taskGroups, taskGroup2)
	}

	job := api.Job{
		Name:        &name,
		Region:      &region,
		Datacenters: []string{"eu-west-1"},
		Type:        &jobType,
		Update:      &updateStrategy,
		TaskGroups:  taskGroups,
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

func createTaskGroup(ctx context.Context, cfg *config.Configuration, name string, groupName string, details *message.Groups, healthcheck *message.Healthcheck,
	revision string) (*api.TaskGroup, error) {

	if groupName != "web" && groupName != "publishing" {
		err := errors.New("Not a valid group name")
		log.Event(ctx, err.Error(), log.ERROR, log.Data{"group_name": groupName})
		return nil, err
	}

	task := createTask(cfg, name+"-"+groupName, details, revision)

	var constraints []*api.Constraint

	if details.DistinctHosts {
		constraint1 := createConstraint("", details.DistinctHosts)
		constraints = append(constraints, &constraint1)
	}

	if details.Mount {
		groupName = groupName + "-mount"
	}
	constraint2 := createConstraint(groupName, false)
	constraints = append(constraints, &constraint2)

	service := createService(name, groupName, healthcheck)

	taskGroup := api.TaskGroup{
		Name:        &groupName,
		Count:       &details.TaskCount,
		Tasks:       []*api.Task{&task},
		Services:    []*api.Service{&service},
		Constraints: constraints,
	}

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
