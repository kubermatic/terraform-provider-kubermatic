provider kubermatic {
  host = "set_it_up"
}
resource "kubermatic_project" "example_project" {
  name = var.project_name
}
resource "kubermatic_cluster" "example_cluster" {
  name       = var.cluster_name
  dc_name    = "set_it_up"
  project_id = kubermatic_project.example_project.id
  credential = "loodse"
  spec {
    version = "1.17.9"
    cloud {
      openstack {
        floating_ip_pool = ""
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