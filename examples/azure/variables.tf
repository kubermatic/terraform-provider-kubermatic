variable "project_name" {
  description = "Name of the MetaKube project"
  default     = "exampleProject"
}

variable "cluster_name" {
  description = "Name of the MetaKube cluster"
  default     = "exampleCluster"
}

variable "k8s_version" {
  description = "The Kubernetes version"
  default     = "1.17.9"
}
