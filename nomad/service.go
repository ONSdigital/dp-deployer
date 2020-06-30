package nomad

import (
	"time"

	"github.com/ONSdigital/dp-deployer/message"
	"github.com/hashicorp/nomad/api"
)

func createService(name string, groupName string, healthcheck *message.Healthcheck) api.Service {
	service := api.Service{
		Name:      name,
		PortLabel: "http",
		Tags:      []string{groupName},
	}

	createServiceCheck(healthcheck)

	return service
}

func createServiceCheck(healthcheck *message.Healthcheck) api.ServiceCheck {
	return api.ServiceCheck{
		Type:     "http",
		Path:     healthcheck.Path,
		Interval: time.Second * 10,
		Timeout:  time.Second * 2,
	}
}
