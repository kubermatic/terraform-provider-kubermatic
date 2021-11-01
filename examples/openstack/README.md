# OpenStack Example

This examples shows how to create Kubermatic project and deploy Kubernetes cluster on OpenStack using Terraform.

To run, set up your OpenStack provider credentials. Configure Kubermatic host address.

Running the example

run `terraform apply` to see it work.

## Run TF
Adjust the variables in `variables.tfvars` and run:
```
terraform plan -var-file variables.tfvars -out .terraform/plan.tf
terraform apply ".terraform/plan.tf"
```
