variable project_name {
  description = "Name of the MetaKube project"
  type        = string
}
variable cluster_name {
  description = "Name of the MetaKube cluster"
  type        = string
}

variable k8s_version {
  description = "Kubernetes version"
  type        = string
}

variable dc_name {
  description = "Datacenter where the cluster should be deployed to. Get a list of possible option with `curl -s -H \"authorization: Bearer $METAKUBE_TOKEN\" https://metakube.syseleven.de/api/v1/dc | jq -r '.[] | select(.seed!=true) | .metadata.name'`"
  type        = string
  default     = "syseleven-dbl1"
}

variable floating_ip_pool {
  description = "Floating IP pool to use for all worker nodes"
  type        = string
  default     = "ext-net"
}

variable username {
  description = "OpenStack username"
  type        = string
}

variable password {
  description = "OpenStack password"
  type        = string
}

variable tenant {
  description = "OpenStack tenant"
  type        = string
}

variable node_deployment_name {
  description = "Name of the MetaKube project"
  type        = string
}

variable node_flavor {
  description = "Flavor of the k8s worker node"
  type        = string
  default     = "m1.medium"
}

variable node_image {
  description = "Name of the image which should be used for worker nodes"
  type        = string
  default     = null
}

variable node_replicas {
  description = "Amount of worker nodes in this node deployment"
  type        = string
  default     = 3
}

variable use_floating_ip {
  description = "If the node deployment should use floating IPs"
  type        = bool
  default     = true
}
