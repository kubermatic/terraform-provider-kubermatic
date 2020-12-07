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
  default     = "1.18.8"
}

variable "azure_client_id" {
  description = "Azure Client ID"
  type        = string
}

variable "azure_subscription_id" {
  description = "Azure Subscription ID"
  type        = string
}

variable "azure_tenant_id" {
  description = "Azure Tenant ID"
  type        = string
}

variable "azure_client_secret" {
  description = "Azure Client Secret"
  type        = string
}