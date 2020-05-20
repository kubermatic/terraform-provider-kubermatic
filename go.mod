module github.com/kubermatic/terraform-provider-kubermatic

go 1.12

replace (
	k8s.io/api => k8s.io/api v0.0.0-20190918195907-bd6ac527cfd2
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190918200256-06eb1244587a
)

require (
	github.com/go-openapi/runtime v0.19.11
	github.com/go-openapi/validate v0.19.5 // indirect
	github.com/google/go-cmp v0.3.1
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/terraform-plugin-sdk v1.6.0
	github.com/kubermatic/go-kubermatic v0.0.0-20200520074811-6441307c58e6
	github.com/mitchellh/go-homedir v1.1.0
	go.uber.org/atomic v1.6.0 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.9.1
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
)
