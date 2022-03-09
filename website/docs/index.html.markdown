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
the last option is not recommended due to possible secret leaking. A new token can get
generated at the KKP UI, see [KKP Documentation > Using Service Accounts](https://docs.kubermatic.com/kubermatic/v2.19/architecture/concept/kkp-concepts/service_account/using_service_account/).

## Argument Reference

The following arguments are supported:

* `host` - (Optional) The hostname (in form of URI) of Kubermatic API. Can be sourced from `KUBERMATIC_HOST`.
* `token` - (Optional) Authentication token. Can be sourced from `KUBERMATIC_TOKEN`.
* `token_path` - (Optional) Path to the kubermatic token. Defaults to `~/.kubermatic/auth`. Can be sourced from `KUBERMATIC_TOKEN_PATH`.
* `log_path` - (Optional) Location to store provider logs. Can be sourced from `KUBERMATIC_LOG_PATH`
* `debug` - (Optional) Set logger to debug level. Can be sourced from `KUBERMATIC_DEBUG`.
* `development` - (Optional) Run development mode. Useful only for contributors. Can be sourced from `KUBERMATIC_DEV`.

## Known Limitations

Please be aware the Kubermatic terraform provider isn't feature complete at the moment and is under development. 
Currently only the following cloud providers are supported with limited functionality (for more details take a look
at the [examples](https://github.com/kubermatic/terraform-provider-kubermatic/tree/master/examples)):
* AWS
* Azure
* OpenStack

If you interested in new features or want to contribute, please take a look at the [open issues](https://github.com/kubermatic/terraform-provider-kubermatic/issues) or the [CONTRIBUTING.md](https://github.com/kubermatic/terraform-provider-kubermatic/blob/master/CONTRIBUTING.md). 
