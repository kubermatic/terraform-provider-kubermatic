PKG_NAME=kubermatic

export GOPATH?=$(shell go env GOPATH)
export GOPROXY=https://proxy.golang.org
export GO111MODULE=on

.PHONY: all
all: install

.PHONY: build
build: fmtcheck bin/terraform-provider-kubermatic

bin/terraform-provider-kubermatic:
	go build -v -o $@

.PHONY: install
install: fmtcheck
	go install

.PHONY: test
test: fmtcheck
	go test ./$(PKG_NAME)

.PHONY: vet
vet:
	@go vet $$(go list ./... | grep -v vendor/) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

.PHONY: fmt
fmt:
	@echo "==> Fixing source code with gofmt..."
	gofmt -s -w ./$(PKG_NAME)

.PHONY: fmtcheck
fmtcheck:
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"
