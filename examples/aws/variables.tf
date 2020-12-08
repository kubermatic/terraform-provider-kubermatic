variable "project_name" {
  description = "Name of the MetaKube project"
  type        = string
}

variable "cluster_name" {
  description = "Name of the MetaKube cluster"
  type        = string
}

variable "k8s_version" {
  description = "The Kubernetes version"
  type        = string
}

variable "aws_access_key_id" {
  description = "AWS Access Key ID"
  type        = string
}

variable "aws_secret_access_key" {
  description = "AWS Access Key Secret"
  type        = string
}

variable "aws_subnet_id" {
  description = "AWS Subnet ID"
  type        = string
}