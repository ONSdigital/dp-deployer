job "dp-deployer" {
  datacenters = ["eu-west-1"]
  region      = "eu"
  type        = "service"

  constraint {
    attribute = "${node.class}"
    value     = "management"
  }

  group "management" {
    count = 1

    restart {
      attempts = 3
      delay    = "15s"
      interval = "1m"
      mode     = "delay"
    }

    task "dp-deployer" {
      driver = "exec"

      artifact {
        source = "s3::https://s3-eu-west-1.amazonaws.com/ons-dp-deployments/dp-deployer/latest.tar.gz"
      }

      config {
        command = "${NOMAD_TASK_DIR}/start-task"

        args = [
          "${NOMAD_TASK_DIR}/dp-deployer",
        ]
      }

      service {
        name = "dp-deployer"
        tags = ["management"]
      }

      resources {
        cpu    = 500
        memory = 512
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
