PKG_NAME=kubermatic
SWEEP_DIR?=./kubermatic
SWEEP?=all

export GOPATH?=$(shell go env GOPATH)
export GOPROXY=https://proxy.golang.org
export GO111MODULE=on

default: install

build: fmtcheck bin/terraform-provider-kubermatic

bin/terraform-provider-kubermatic:
	go build -v -o $@

install: fmtcheck
	go install

test: fmtcheck
	go test $(PKG_NAME)

testacc:
	TF_ACC=1 go test $(PKG_NAME) -v $(TESTARGS) -timeout 120m

testacc:
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
