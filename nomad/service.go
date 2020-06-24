package nomad

import "github.com/hashicorp/nomad/api"

func CreateService(tags []string, path string) {
	service := api.Service {
		Name: serviceName,
		PortLabel: "http",
		Tags: [], // web or publishing
	}

	createServiceCheck(path)
}

func createServiceCheck() {
	serviceCheck := api.ServiceCheck {
		Type: "http",
		Path: ,
		Interval: "10s",
		Timeout: "2s"
	}
}