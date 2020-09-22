variable "project_name" {
  description = "Name of the Kubermatic project"
  default     = "exampleProject"
}

variable "cluster_name" {
  description = "Name of the Kubermatic cluster"
  default     = "exampleCluster"
}

variable "k8s_version" {
  description = "The Kubernetes version"
  default     = "1.17.9"
}