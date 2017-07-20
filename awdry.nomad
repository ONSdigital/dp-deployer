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

    task "runner" {
      driver = "exec"

      artifact {
        source = "s3::https://s3-eu-west-1.amazonaws.com/ons-dp-deployments/awdry/latest.tar.gz"
      }

      config {
        command = "${NOMAD_TASK_DIR}/scripts/start"
      }

      service {
        name = "awdry"
        tags = ["management"]
      }

      resources {
        cpu    = 500
        memory = 512
      }
    }
  }
}
