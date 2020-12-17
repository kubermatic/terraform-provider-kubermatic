terraform {
  required_providers {
    metakube = {
      source  = "syseleven/metakube"
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

// You can add as many collaborators as you want.
//  user {
//    email = "FILL_IN"
//    group = "owners" // editors, viewers
//  }
}

data "local_file" "public_sshkey" {
  filename = pathexpand(var.public_sshkey_file)
}

resource "metakube_sshkey" "local" {
  project_id = metakube_project.project.id

  name = "local SSH key"
  public_key = data.local_file.public_sshkey.content
}

resource metakube_cluster "cluster" {
  name       = var.cluster_name
  dc_name    = var.dc_name
  project_id = metakube_project.project.id
  sshkeys = [metakube_sshkey.local.id]

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

# create admin.conf file
resource "local_file" "kubeconfig" {
  content     = metakube_cluster.cluster.kube_config
  filename = "${path.module}/admin.conf"
}

resource metakube_node_deployment "node_deployment" {
  name       = null // auto generate
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