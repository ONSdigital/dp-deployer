package nomad

import (
	"time"

	"github.com/ONSdigital/dp-deployer/config"
	"github.com/ONSdigital/dp-deployer/message"
	"github.com/hashicorp/nomad/api"
)

func createTask(cfg *config.Configuration, name string, details *message.Groups, revision string) api.Task {
	config := make(map[string]interface{})
	portMap := make(map[string]interface{})

	portMap["http"] = "${NOMAD_PORT_http}"
	config["command"] = "${NOMAD_TASK_DIR}/start-task"
	config["args"] = details.CommandLineArgs
	config["image"] = cfg.ECR_URL + ":concourse-" + revision
	config["port_map"] = portMap
	config["volumes"] = details.Volumes
	config["userns_mode"] = details.UsernsMode

	resources := createResources(details)
	restartPolicy := createRestartPolicy()
	vault := createVault(name)
	template := createTemplate()
	artifact := createTaskArtifact(cfg)

	task := api.Task{
		Name:          name,
		Driver:        "docker",
		Config:        config,
		Resources:     &resources,
		RestartPolicy: &restartPolicy,
		Vault:         &vault,
		Templates:     []*api.Template{&template},
		Artifacts:     []*api.TaskArtifact{&artifact},
	}

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

func createVault(name string) api.Vault {
	return api.Vault{
		Policies: []string{name},
	}
}

func createResources(details *message.Groups) api.Resources {

	networkResources := createNetworkResources()

	return api.Resources{
		CPU:      &details.CPU,
		MemoryMB: &details.Memory,
		Networks: []*api.NetworkResource{&networkResources},
	}

}

func createNetworkResources() api.NetworkResource {
	return api.NetworkResource{
		DynamicPorts: []api.Port{{Label: "http"}},
	}
}

// TODO rename genericthing.zip when we know what it will be called.
func createTaskArtifact(cfg *config.Configuration) api.TaskArtifact {
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
