variable "project_id" {
  description = "ID of the Kubermatic project"
  validation {
    condition = length(var.project_id) == 10
    error_message = "No valid project_id provided."
  }
}

variable "cluster_name" {
  description = "Name of the Kubermatic cluster"
  validation {
    condition = length(var.cluster_name) >= 5
    error_message = "Cluster name must have at least 5 characters."
  }
}

variable "dc_name" {
  description = "Name of the datacenter where the cluster gets provisioned."

  validation {
    condition = length(var.dc_name) > 0
    error_message = "You must provide a valid datacenter."
  }
}

variable "public_ssh_key" {
  description = "SSH Key to be added to the Machine Deployments"
  default = null
}

variable "k8s_cp_version" {
  description = "The Kubernetes control plane version"
  default     = "1.21.5"
}

# OS Settings
variable "floating_ip_pool" {
  description = "The floating IP pool to be used."
  default = "public"
}

variable "preset" {
  description = "The preset containing the OS credentials."
}

# MachineDeployment
variable "md_replicas" {
  description = "Number of machines in the machine deployment."
  default = 0
}

variable "md_flavor" {
  description = "OS Flavor to be used for the machines."
  default = "m1.medium"
}

variable "md_image" {
  description = "Image to be used for the machines."
}

variable "k8s_md_version" {
  description = "The Kubernetes worker node version"
  default = "1.21.5"
}

variable "instance_ready_check_period" {
  // Don't lower this value!!!
  default = "5s"
}

variable "instance_ready_check_timeout" {
  // Don't lower this value!!!
  default = "120s"
}
