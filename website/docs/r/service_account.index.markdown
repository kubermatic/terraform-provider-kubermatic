---
layout: "metakube"
page_title: "MetaKube: metakube_service_account"
sidebar_current: "docs-metakube-service-account"
description: |-
  Service account resource in the Terraform provider metakube.
---

# metakube_resource

Service account resource in the provider defines the corresponding service account in MetaKube.

## Example usage

```hcl
resource "metakube_service_account" "acctest_sa" {
	project_id = "project id"
	name = "dev account"
	group = "viewers"
}
```

## Argument reference

The following arguments are supported:

* `project_id` - (Required) ID of a project to add service account to.
* `name` - (Required) Service account's name.
* `group` - (Required) Service account's role in the project.
