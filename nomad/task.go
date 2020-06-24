package nomad

import (
	"github.com/hashicorp/nomad/api"
)

func createTask(taskName string, driver string, configMap map[string]interface{}) {
	task := api.Task {
		Name: taskName,
		Driver: "docker",
		Config: [],
	}
}