terraform {
  required_providers {
    metakube = {
      source  = "syseleven/metakube"
      version = "0.1.1"
    }
    openstack = {
      source = "terraform-provider-openstack/openstack"
    }
  }
}

data openstack_images_image_v2 "image" {
  most_recent = true

  visibility = "public"
  properties = {
    os_distro  = "ubuntu"
    os_version = "18.04"
  }
}

provider metakube {}
resource metakube_project "project" {
  name = var.project_name
}

resource metakube_cluster "cluster" {
  name       = var.cluster_name
  dc_name    = var.dc_name
  project_id = metakube_project.project.id
  spec {
    version = var.k8s_version
    cloud {
      openstack {
        floating_ip_pool = var.floating_ip_pool
        password         = var.password
        tenant           = var.tenant
        username         = var.username
      }
    }
  }
}
resource metakube_node_deployment "node_deployment" {
  name       = var.node_deployment_name
  cluster_id = metakube_cluster.cluster.id
  spec {
    replicas = var.node_replicas
    template {
      cloud {
        openstack {
          flavor          = var.node_flavor
          image           = var.node_image != null ? var.node_image : data.openstack_images_image_v2.image.name
          use_floating_ip = var.use_floating_ip
        }
      }
      operating_system {
        ubuntu {
        }
      }
      versions {
        kubelet = var.k8s_version
      }
    }
  }
}
