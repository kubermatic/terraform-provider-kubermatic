---
layout: "kubermatic"
page_title: "Kubermatic: kubermatic_project"
sidebar_current: "docs-kubermatic-project"
description: |-
  Project resource in the Terraform provider kubermatic.
---

# kubermatic_resource

Project resource in the provider defines the corresponding project in Kubermatic.

## Example Usage

```hcl
resource "kubermatic_project" "example" {
  name = "example"
  labels = {
    "foo": "bar"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Project name.
* `labels` - (Optional) Project labels.
* `user` - (Optional) Set of users assigned to the project.

## Attributes

* `status` - The current state of the project.
* `creation_timestamp` - Timestamp of resource creation.
* `deletion_timestamp` - Timestamp of resource deletion.

## Nested blocks

### `user`

#### Arguments

* `email` - (Required) User's email address.
* `group` - (Required) User's role in the project.
