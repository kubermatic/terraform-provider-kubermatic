PKG_NAME=kubermatic
SWEEP_DIR?=./kubermatic
SWEEP?=all

export GOPATH?=$(shell go env GOPATH)
export GOPROXY=https://proxy.golang.org
export GO111MODULE=on
export GOOS=$(shell go env GOOS)

default: install

build: fmtcheck bin/terraform-provider-kubermatic

bin/terraform-provider-kubermatic:
	go build -v -o $@

install: fmtcheck
	go install

test: fmtcheck
	go test ./$(PKG_NAME)

testacc:
# Require following environment variables to be set:
# KUBERMATIC_TOKEN - access token
# KUBERMATIC_HOST - example https://kubermatic.io
# KUBERMATIC_ANOTHER_USER_EMAIL - email of an existing user to test cluster access sharing
# KUBERMATIC_K8S_VERSION - the kubernetes version
# KUBERMATIC_K8S_OLDER_VERSION - lower kubernetes version then KUBERMATIC_K8S_VERSION
# KUBERMATIC_OPENSTACK_IMAGE - an image available for openstack clusters
# KUBERMATIC_OPENSTACK_IMAGE2 - another image available for openstack clusters
# KUBERMATIC_OPENSTACK_FLAVOR - openstack flavor to use
# KUBERMATIC_OPENSTACK_USERNAME - openstack credentials username
# KUBERMATIC_OPENSTACK_PASSWORD - openstack credentials password
# KUBERMATIC_OPENSTACK_TENANT - openstack tenant to use
# KUBERMATIC_OPENSTACK_NODE_DC - openstack node datacenter name
# KUBERMATIC_AZURE_NODE_DC - azure node datacenter name
# KUBERMATIC_AZURE_NODE_SIZE
# KUBERMATIC_AZURE_CLIENT_ID
# KUBERMATIC_AZURE_CLIENT_SECRET
# KUBERMATIC_AZURE_TENANT_ID
# KUBERMATIC_AZURE_SUBSCRIPTION_ID
# KUBERMATIC_AWS_ACCESS_KEY_ID
# KUBERMATIC_AWS_ACCESS_KEY_SECRET
# KUBERMATIC_AWS_VPC_ID
# KUBERMATIC_AWS_NODE_DC
# KUBERMATIC_AWS_INSTANCE_TYPE
# KUBERMATIC_AWS_SUBNET_ID
# KUBERMATIC_AWS_AVAILABILITY_ZONE
# KUBERMATIC_AWS_DISK_SIZE
	TF_ACC=1 go test ./$(PKG_NAME) -v $(TESTARGS) -timeout 120m

sweep:
	@echo "WARNING: This will destroy infrastructure. Use only in development accounts."
	go test $(SWEEP_DIR) -v -sweep=$(SWEEP) $(SWEEPARGS) -timeout 60m

vet:
	@go vet $$(go list ./... | grep -v vendor/) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

fmt:
	@echo "==> Fixing source code with gofmt..."
	gofmt -s -w $(PKG_NAME)

fmtcheck:
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

.PHONY: build install test testacc sweep vet fmt fmtcheck
