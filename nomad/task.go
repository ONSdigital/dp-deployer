package nomad

import (
	"time"

	"github.com/hashicorp/nomad/api"
)

func createTask(taskName string, args []string, volumes []string, userns_mode string) api.Task {
	config := make(map[string]interface{})
	portMap := make(map[string]interface{})

	portMap["http"] = ""
	config["command"] = "${NOMAD_TASK_DIR}/start-task"
	config["args"] = args
	config["image"] = "{{ECR_URL}}:concourse-{{REVISION}}"
	config["port_map"] = portMap
	config["volumes"] = volumes
	config["userns_mode"] = userns_mode

	task := api.Task{
		Name:   taskName,
		Driver: "docker",
		Config: config,
	}

	createRestartPolicy()

	return task
}

func createRestartPolicy() api.RestartPolicy {
	attempts := 3
	delay := time.Second * 15
	interval := time.Second * 60
	mode := "delay"

	return api.RestartPolicy{
		Attempts: &attempts,
		Delay:    &delay,
		Interval: &interval,
		Mode:     &mode,
	}
}

func createTaskArtifact() api.TaskArtifact {
	source := "s3::https://s3-eu-west-1.amazonaws.com/{{DEPLOYMENT_BUCKET}}/genericthing.zip"

	return api.TaskArtifact{
		GetterSource: &source,
	}

}
