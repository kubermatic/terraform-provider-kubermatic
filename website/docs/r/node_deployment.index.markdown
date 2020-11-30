---
layout: "metakube"
page_title: "MetaKube: metakube_node_deployment"
sidebar_current: "docs-metakube-node-deployment"
description: |-
  Node deployment resource in the Terraform provider metakube.
---

# metakube_resource

Node deployment resource in the provider defines the corresponding deployment of nodes.

## Example usage

```hcl
resource "metakube_node_deployment" "example_node" {
  cluster_id = metakube_project.example_project.id + ":europe-west3-c:" + metakube_cluster.example_cluster.id
  spec {
    replicas = 1
    template {
      cloud {
        aws {
          instance_type     = "t3.small"
          disk_size         = 25
          volume_type       = "standard"
          subnet_id         = "subnet-04f2f551bbc697db3"
          availability_zone = "eu-central-1c"
          assign_public_ip  = true
        }
      }
      operating_system {
        ubuntu {
          dist_upgrade_on_boot = true
        }
      }
    }
  }
}
```

## Argument reference

The following arguments are supported:

* `cluster_id` - (Required) Reference full cluster identifier of format <project id>:<seed dc>:<cluster id>.
* `name` - (Required) Cluster name.
* `spec` - (Required) Node deployment specification.

## Attributes

* `creation_timestamp` - Timestamp of resource creation.
* `deletion_timestamp` - Timestamp of resource deletion.

## Nested Blocks

### `spec`

#### Arguments

* `replicas` - (Optional) Number of replicas, default = 1.
* `template` - (Required) Template specification.
* `dynamic_config` - (Optional) MetaKube dynamic config.

### `template`

#### Arguments

* `cloud` - (Required) Cloud specification.
* `operating_system` - (Required) Operating system settings.
* `versions` - (Optional) K8s components versions.
* `labels` - (Optional) Map of labels to set on nodes.
* `taints` - (Optional) List of taints to set on nodes.

### `cloud`

One of the following must be selected.

#### Arguments

* `bringyourown` - (Optional) User defined specification.
* `aws` - (Optional) AWS node deployment specification.

### `operating_system`

One of the following must be selected.

#### Arguments

* `ubuntu` - (Optional) Ubuntu operating system and its settings.

### `versions`

#### Arguments

* `kubelet` - (Optional) Kubelet version.

### `taints`

#### Arguments

* `effect` - (Required) Effect for taint. Accepted values are NO_SCHEDULE, PREFER_NO_SCHEDULE, and NO_EXECUTE.
* `key` - (Required) Key for taint.
* `value` - (Required) Value for taint.

### `aws`

#### Arguments

* `instance_type` - (Required) EC2 instance type
* `disk_size` - (Required) Size of the volume in GBs.
* `volume_type` -  (Required) EBS volume type.
* `availability_zone` - (Required) Availability zone in which to place the node. It is coupled with the subnet to which the node will belong.
* `subnet_id` - (Required) The VPC subnet to which the node shall be connected.
* `assign_public_ip` - (Optional) When set the AWS instance will get a public IP address assigned during launch overriding a possible setting in the used AWS subnet.
* `ami` - (Optional) Amazon Machine Image to use. Will be defaulted to an AMI of your selected operating system and region.
* `tags`- (Optional) Additional EC2 instance tags.

### `ubuntu`

#### Arguments

* `dist_upgrade_on_boot` - (Optional) Upgrade operating system on boot, default to false.
