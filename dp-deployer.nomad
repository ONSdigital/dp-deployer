job "dp-deployer" {
  datacenters = ["eu-west-1"]
  region      = "eu"
  type        = "service"

  constraint {
    attribute = "${node.class}"
    value     = "management"
  }

  group "management" {
    count = "{{MANAGEMENT_TASK_COUNT}}"

    restart {
      attempts = 3
      delay    = "15s"
      interval = "1m"
      mode     = "delay"
    }

    task "dp-deployer" {
      driver = "docker"

      artifact {
        source = "s3::https://s3-eu-west-1.amazonaws.com/{{DEPLOYMENT_BUCKET}}/dp-deployer/{{REVISION}}.tar.gz"
      }

      config {
        command = "${NOMAD_TASK_DIR}/start-task"

        args = ["./dp-deployer"]

        image = "{{ECR_URL}}:concourse-{{REVISION}}"

        volumes = [
          "/etc/nomad/tls:/etc/nomad/tls",
        ]
      }

      service {
        name = "dp-deployer"
        tags = ["management"]
      }

      resources {
        cpu    = "{{MANAGEMENT_RESOURCE_CPU}}"
        memory = "{{MANAGEMENT_RESOURCE_MEM}}"
      }

      template {
        source      = "${NOMAD_TASK_DIR}/vars-template"
        destination = "${NOMAD_TASK_DIR}/vars"
      }

      vault {
        policies = ["dp-deployer"]
      }

    }
  }
}
