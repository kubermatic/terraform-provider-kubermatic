# How to Contribute

Loodse projects are [Apache 2.0 licensed](LICENSE) and accept contributions via
GitHub pull requests.  This document outlines some of the conventions on
development workflow, commit message formatting, contact points and other
resources to make it easier to get your contribution accepted.

## Certificate of Origin

By contributing to this project you agree to the Developer Certificate of
Origin (DCO). This document was created by the Linux Kernel community and is a
simple statement that you, as a contributor, have the legal right to make the
contribution. See the [DCO](DCO) file for details.

Any copyright notices in this repo should specify the authors as "the Loodse XXX project contributors".

To sign your work, just add a line like this at the end of your commit message:

```
Signed-off-by: Joe Example <joe@example.com>
```

This can easily be done with the `--signoff` option to `git commit`.

By doing this you state that you can certify the following (from https://developercertificate.org/):

## Email and Chat

The terraform-provider-kubermatic project currently uses the general Loodse email list and Slack channel:
- Email: [loodse-dev](https://groups.google.com/forum/#!forum/loodse-dev)
- Slack: #[Slack](http://slack.kubermatic.io/) on Slack

Please avoid emailing maintainers found in the MAINTAINERS file directly. They
are very busy and read the mailing lists.

##  Reporting a security vulnerability

Due to their public nature, GitHub and mailing lists are not appropriate places for reporting vulnerabilities. If you suspect you have found a security vulnerability in rkt, please do not file a GitHub issue, but instead email security@loodse.com with the full details, including steps to reproduce the issue.


## Getting Started

- Fork the repository on GitHub
- Read the [README](README.md) for build and test instructions
- Play with the project, submit bugs, submit patches!

### Contribution Flow

This is a rough outline of what a contributor's workflow looks like:

- Create a topic branch from where you want to base your work (usually master).
- Make commits of logical units.
- Make sure your commit messages are in the proper format (see below).
- Push your changes to a topic branch in your fork of the repository.
- Make sure the tests pass, and add any new tests as appropriate.
- Submit a pull request to the original repository.

Thanks for your contributions!

### Build Terraform Provider Locally
Test your current changes:
```
make test
make fmt
```

Build the provider:
```
make build
```

Copy the provider to the Terraform plugin directory:
```
export OSTYPE=linux # darwin

mkdir -p ~/.terraform.d/plugins/terraform.example.com/local/kubermatic/0.1.0/${OSTYPE}_amd64
cp bin/terraform-provider-kubermatic ~/.terraform.d/plugins/terraform.example.com/local/kubermatic/0.1.0/${OSTYPE}_amd64/
```
That's it!

Refer in your TF manifests to the specific provider, i.e.:
```
terraform {
  required_providers {
    kubermatic = {
      source = "terraform.example.com/local/kubermatic"
      version = "~> 0.1.0"
    }
  }
}
provider "kubermatic" {
  host  = "https://dev.kubermatic.io"
  token_path = "./token"
}
```
Add a token to `./token`. Either use your personal access token that you get when you go to the `https://dev.kubermatic.io/rest-api` or you create a Service Account in the target project with `Editor` or `ProjectManager` permissions.
