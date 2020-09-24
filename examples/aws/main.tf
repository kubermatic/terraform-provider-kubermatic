provider kubermatic {
  host = "set_it_up"
}
resource "kubermatic_project" "example_project" {
  name = var.project_name
}
resource "kubermatic_cluster" "example_cluster" {
  name       = var.cluster_name
  dc_name    = "aws-eu-central-1a"
  project_id = kubermatic_project.example_project.id
  spec {
    version = var.k8s_version
    cloud {
      aws {
        access_key_id     = "set_it_up"
        secret_access_key = "set_it_up"
      }
    }
  }
}
resource "kubermatic_node_deployment" "example_node" {
  name       = "examplenode"
  cluster_id = kubermatic_cluster.example_cluster.id
  spec {
    replicas = 2
    template {
      cloud {
        aws {
          instance_type     = "t3.small"
          disk_size         = 25
          volume_type       = "standard"
          availability_zone = "eu-central-1c"
          subnet_id         = "set_it_up"
          assign_public_ip  = true

        }
      }
      operating_system {
        ubuntu {
          dist_upgrade_on_boot = false
        }
      }
      versions {
        kubelet = var.k8s_version
      }
    }
  }
}