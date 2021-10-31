provider kubermatic {
  host = "set_it_up"
  token_path= "path"
}

resource "kubermatic_project" "example_project" {
  name = var.project_name
}

# Create Cluster Resource (without MachineDeployment for more-finegrained control)
resource "kubermatic_cluster" "example_cluster" {
  name       = var.cluster_name
  dc_name    = "set_it_up"
  project_id = kubermatic_project.example_project.id
  credential = "preset_name"
  spec {
    version = "1.21.0"
    enable_user_ssh_key_agent = true # (default)
    opa_integration {
      enabled = false # (default)
      webhook_timeout_seconds = 0 # (default)
    }
    mla {
      logging_enabled = true
      monitoring_enabled = true
    }
    cloud {
      openstack {
        floating_ip_pool = "public"
        # == Credentials
        # Not needed if "credential" set.
        password         = "set_it_up"
        tenant           = "set_it_up"
        username         = "set_it_up"
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
        openstack {
          flavor = "l1c.tiny"
          image  = "kubermatic-e2e-ubuntu"
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