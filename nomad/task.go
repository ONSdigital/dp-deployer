package nomad

import (
	"time"

	"github.com/hashicorp/nomad/api"
)

func createTask(name string, args []string, volumes []string, userns_mode string) api.Task {
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
		Name:   name, // will neeed the suffix of web or publishing
		Driver: "docker",
		Config: config,
	}

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
		Policies: []string{name}, // will neeed the suffix of web or publishing
	}
}

func createResources() api.Resources {

	createNetworkResources()

	return api.Resources{
		CPU:    "", // publishing or web
		Memory: "", // publishing or web
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
