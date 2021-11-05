# ========== RESOURCES ==========
# ===============================
resource "kubermatic_sshkey" "test" {
    project_id = var.project_id
    name = "${var.cluster_name}-ssh-key"
    public_key = var.public_ssh_key
}

resource "kubermatic_cluster" "test" {
  project_id = var.project_id
  name = var.cluster_name
  sshkeys = [
      kubermatic_sshkey.test.id
  ]
  dc_name = var.dc_name
  credential = var.preset

  spec {
    version = var.k8s_cp_version
    enable_user_ssh_key_agent = true
    opa_integration {
      enabled = false
      webhook_timeout_seconds = 0 # (default)
    }
    cloud {
      openstack {
        floating_ip_pool = var.floating_ip_pool
      }
    }
  }
}

resource "kubermatic_node_deployment" "test" {
  name       = "${var.cluster_name}-nodedeployment"
  project_id = var.project_id
  dc_name = var.dc_name
  cluster_id = kubermatic_cluster.test.id

  spec {
    replicas = var.md_replicas
    template {
      cloud {
        openstack {
          flavor = var.md_flavor
          image  = var.md_image
          instance_ready_check_period = var.instance_ready_check_period
          instance_ready_check_timeout = var.instance_ready_check_timeout
        }
      }
      operating_system {
        flatcar {
          disable_auto_update = false
        }
      }
      versions {
        kubelet = var.k8s_md_version
      }
    }
  }
}

# ========= DATA SOURCE =========
# ===============================

data "kubermatic_cluster_kubeconfig" "test" {
  project_id = kubermatic_cluster.test.project_id
  cluster_id = kubermatic_cluster.test.id
}

data "kubermatic_cluster" "test" {
  project_id = kubermatic_cluster.test.project_id
  cluster_id = kubermatic_cluster.test.id
}

data "kubermatic_node_deployment" "test" {
  project_id = kubermatic_node_deployment.test.project_id
  cluster_id = kubermatic_node_deployment.test.cluster_id
  dc_name    = kubermatic_node_deployment.test.dc_name
  id         = kubermatic_node_deployment.test.id
}


# =========== OUTPUT ============
# ===============================
output "project" {
  value = var.project_id
}

output "kubeconfig" {
  value = data.kubermatic_cluster_kubeconfig.test.kubeconfig
}

output "cluster" {
  value = data.kubermatic_cluster.test
}

output "node_deployment" {
  value = data.kubermatic_node_deployment.test
}
