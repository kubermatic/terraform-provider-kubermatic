# service_account_token Resource

Service account token resource in the provider defines the corresponding resource in MetaKube.

## Example usage

```hcl
resource "metakube_service_account_token" "acctest_sa" {
	project_id = "project id"
	name = "dev account"
	group = "viewers"
}
```

## Argument reference

The following arguments are supported:

* `service_account_id` - (Required) Service account full identifier of format `project_id:service_account_id`.
* `name` - (Required) Name for the token.

## Attributes
* `token` - (Required) Token value.
* `expiry` - Expiration timestamp.
* `creation_timestamp` - Timestamp of resource creation.
