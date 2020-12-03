---
layout: "metakube"
page_title: "MetaKube: metakube_sshkey"
sidebar_current: "docs-metakube-sshkey"
description: |-
  SSH Key resource in the Terraform provider metakube.
---

# metakube_resource

SSH Key resource in the provider defines the corresponding sshkey with public key in MetaKube.

## Example Usage

```hcl
resource "metakube_sshkey" "example" {
  project_id = metakube_project.example.id
  name = "example"
  public_key = "ssh-rsa ... foo@bar.net"
}
```

## Argument Reference

The following arguments are supported:

* `project_id` - (Required) Reference project identifier.
* `name` - (Required) Name for the resource.
* `public_key` - (Required) Public ssh key.
