provider metakube {
  host = "set_it_up"
}
resource "metakube_project" "example_project" {
  name = var.project_name
}
resource "metakube_cluster" "example_cluster" {
  name       = var.cluster_name
  dc_name    = "azure-westeurope"
  project_id = metakube_project.example_project.id
  spec {
    version = var.k8s_version
    cloud {
      azure {
        client_id       = "set_it_up"
        subscription_id = "set_it_up"
        tenant_id       = "set_it_up"
        client_secret   = "set_it_up"
      }
    }
  }
}
resource "metakube_node_deployment" "example_node" {
  name       = "examplenode"
  cluster_id = metakube_cluster.example_cluster.id
  spec {
    replicas = 2
    template {
      cloud {
        azure {
          size = "Standard_D1_v2"
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
