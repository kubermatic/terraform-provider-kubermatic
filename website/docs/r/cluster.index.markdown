---
layout: "kubermatic"
page_title: "Kubermatic: kubermatic_cluster"
sidebar_current: "docs-kubermatic-cluster"
description: |-
  Cluster resource in the Terraform provider kubermatic.
---

# kubermatic_resource

Cluster resource in the provider defines the corresponding cluster in Kubermatic.

## Example Usage

```hcl
resource "kubermatic_cluster" "example" {
  project_id = kubermatic_project.example.id
  name = "example"
  dc_name = "europe-west3-c"
  credential = "loodse"

  spec {
    version = "1.15.6"
    cloud {

      aws {
        instance_profile_name = "example-profile-name"
      }

      dc = "aws-eu-central-1a"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `project_id` - (Required) Reference project identifier.
* `dc_name` - (Required) Data center name.
* `name` - (Required) Cluster name.
* `spec` - (Required) Cluster specification.
* `labels` - (Optional) Labels added to cluster.
* `sshkeys` - (Optional) SSH keys attached to nodes. 
* `credential` - (Optional) Cluster access credentials.
* `type` - (Optional) Cloud orchestrator, either Kubernetes or OpenShift.

## Attributes

* `creation_timestamp` - Timestamp of resource creation.
* `deletion_timestamp` - Timestamp of resource deletion.

## Nested Blocks

### `spec`

#### Arguments

* `version` - (Required) Cloud orchestrator version, either Kubernetes or OpenShift.
* `cloud` - (Required) Cloud provider specification.
* `machine_networks` - (Optional) Machine networks, optionally specifies the parameters for IPAM.
* `audit_logging` - (Optional) Audit logging settings.

### `cloud`

One of the following must be selected.

#### Arguments

* `bringyourown` - (Optional) User defined infrastructure.
* `aws` - (Optional) Amazon Web Services infrastructure.

### `aws`

#### Arguments

* `access_key_id` - (Required) Access key id, can be passed as AWS_ACCESS_KEY_ID env.
* `secret_access_key` - (Required) Secret access key, can be passed as AWS_SECRET_ACCESS_KEY env.
* `vpc_id` - (Optional) Virtual private cloud identifier.
* `security_group_id` - (Optional) Security group identifier.
* `route_table_id` - (Optional) Route table identifier.
* `instance_profile_name` - (Optional) Instance profile name.
* `role_arn` - (Optional) The IAM role that the control plane will use.
