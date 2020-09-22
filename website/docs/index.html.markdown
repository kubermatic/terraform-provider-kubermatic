---
layout: "kubermatic"
page_title: "Provider: Kubermatic"
sidebar_current: "docs-kubermatic-index"
description: |-
  Terraform provider kubermatic.
---

# Kubermatic Provider

The Kubermatic provider is used to interact with the resources supported by Kubermatic.
The provider needs to be configured with the proper auth token (`~/.kubermatic/auth`) before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
provider "kubermatic" {
  host = "https://kubermatic-api-address"
}

# Example project configuration
resource "kubermatic_project" "project" {
  name = "terraform-project"
}
```

## Authentication

The provider tries to read a token from `~/.kubermatic/auth` by default,
it is possible to change the token location by setting `token_path` argument.
Another way of authentication is to pass `KUBERMATIC_TOKEN` env or set `token` param,
the last option is not recommended due to possible secret leaking.

## Argument Reference

The following arguments are supported:

* `host` - (Optional) The hostname (in form of URI) of Kubermatic API. Can be sourced from `KUBERMATIC_HOST`.
* `token` - (Optional) Authentication token. Can be sourced from `KUBERMATIC_TOKEN`.
* `token_path` - (Optional) Path to the kubermatic token. Defaults to `~/.kubermatic/auth`. Can be sourced from `KUBERMATIC_TOKEN_PATH`.
* `log_path` - (Optional) Location to store provider logs. Can be sourced from `KUBERMATIC_LOG_PATH`
* `debug` - (Optional) Set logger to debug level. Can be sourced from `KUBERMATIC_DEBUG`.
* `development` - (Optional) Run development mode. Useful only for contributors. Can be sourced from `KUBERMATIC_DEV`.
