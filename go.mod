module github.com/syseleven/terraform-provider-metakube

go 1.15

replace (
	k8s.io/api => k8s.io/api v0.0.0-20190918195907-bd6ac527cfd2
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190918200256-06eb1244587a
)

require (
	github.com/go-openapi/errors v0.20.0 // indirect
	github.com/go-openapi/runtime v0.19.27
	github.com/go-openapi/strfmt v0.20.1
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/go-openapi/validate v0.20.2 // indirect
	github.com/google/go-cmp v0.5.4
	github.com/hashicorp/go-cty v1.4.1-0.20200414143053-d3edf31b6320
	github.com/hashicorp/go-version v1.2.1
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.5.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/syseleven/go-metakube v0.0.0-20210416091829-b6a6fb1490a6
	go.uber.org/zap v1.16.0
)
