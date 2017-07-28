job "awdry" {
  datacenters = ["eu-west-1"]
  region      = "eu"
  type        = "service"

  constraint {
    attribute = "${node.class}"
    value     = "management"
  }

  group "management" {
    count = 1

    task "awdry" {
      driver = "exec"

      artifact {
        source = "s3::https://s3-eu-west-1.amazonaws.com/ons-dp-deployments/awdry/latest.tar.gz"
      }

      config {
        command = "${NOMAD_TASK_DIR}/start-task"

        args = [
          "${NOMAD_TASK_DIR}/awdry",
        ]
      }

      service {
        name = "awdry"
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
        policies = ["awdry"]
      }
    }
  }
}
