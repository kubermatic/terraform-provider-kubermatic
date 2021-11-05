# ======== Terraform Settings
terraform {
  required_providers {
    kubermatic = {
      source = "kubermatic/kubermatic"
      version = "0.2.0"
    }
  }
}

provider "kubermatic" {
  host  = "https://dev.kubermatic.io"
  token_path = "./token"
}
# ================================
