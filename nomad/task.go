package nomad

import (
	"time"

	"github.com/ONSdigital/dp-deployer/message"
	"github.com/hashicorp/nomad/api"
)

func createTask(name string, details *message.Groups) api.Task {
	config := make(map[string]interface{})
	portMap := make(map[string]interface{})

	portMap["http"] = ""
	config["command"] = "${NOMAD_TASK_DIR}/start-task"
	config["args"] = details.CommandLineArgs
	config["image"] = "{{ECR_URL}}:concourse-{{REVISION}}"
	config["port_map"] = portMap
	config["volumes"] = details.Volumes
	config["userns_mode"] = details.UsernsMode

	task := api.Task{
		Name:   name,
		Driver: "docker",
		Config: config,
	}

	createResources(details)
	createRestartPolicy()
	CreateVault(name)

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

func CreateVault(name string) api.Vault {
	return api.Vault{
		Policies: []string{name},
	}
}

func createResources(details *message.Groups) api.Resources {

	createNetworkResources()

	return api.Resources{
		CPU:      &details.CPU,
		MemoryMB: &details.Memory,
	}

}

func createNetworkResources() api.NetworkResource {
	return api.NetworkResource{
		DynamicPorts: []api.Port{api.Port{Label: "http"}},
	}
}

func createTaskArtifact() api.TaskArtifact {
	source := "s3::https://s3-eu-west-1.amazonaws.com/" + cfg.DeploymentsBucketName + "/genericthing.zip"

	return api.TaskArtifact{
		GetterSource: &source,
	}

}

func createTemplate() api.Template {
	source := "${NOMAD_TASK_DIR}/vars-template"
	destination := "${NOMAD_TASK_DIR}/vars"
	return api.Template{
		SourcePath: &source,
		DestPath:   &destination,
	}
}
