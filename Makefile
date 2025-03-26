# Detect platform for sed compatibility
SED := $(shell if [ "$(shell uname)" = "Darwin" ]; then echo gsed; else echo sed; fi)

# VERSION defines the project version, extracted from cmd/main.go without leading 'v'.
VERSION := $(shell awk -F'"' '/const version/{gsub(/^v/, "", $$2); print $$2}' cmd/main.go)

# ENVTEST_K8S_VERSION refers to the version of kbebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.30.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set).
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

## Detect platform for Kind binary
UNAME := $(shell uname -s | tr '[:upper:]' '[:lower:]')
KIND_BINARY := kind-$(UNAME)-amd64

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KIND = $(LOCALBIN)/kind
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint

## Tool Versions
# renovate: datasource=github-releases depName=kubernetes-sigs/controller-runtime
ENVTEST_VERSION ?= release-0.18
# renovate: datasource=github-releases depName=golangci/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.0.2
# renovate: datasource=github-releases depName=google/yamlfmt
YAMLFMT_VERSION ?= v0.16.0
# renovate: datasource=github-releases depName=kubernetes-sigs/kind
KIND_VERSION ?= 0.27.0

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Versioning

.PHONY: patch
patch: ## Increment the patch version (x.y.Z -> x.y.(Z+1)).
	@NEW_VERSION=$(shell echo $(VERSION) | awk -F. '{print $$1"."$$2"."$$3+1}') && \
	$(SED) -i -E "s/(const version string = \"v)[^\"]+/\1$${NEW_VERSION}/" cmd/main.go && \
	$(MAKE) update-version VERSION=$${NEW_VERSION}

.PHONY: minor
minor: ## Increment the minor version (x.Y.z -> x.(Y+1).0).
	@NEW_VERSION=$(shell echo $(VERSION) | awk -F. '{print $$1"."$$2+1".0"}') && \
	$(SED) -i -E "s/(const version string = \"v)[^\"]+/\1$${NEW_VERSION}/" cmd/main.go && \
	$(MAKE) update-version VERSION=$${NEW_VERSION}

.PHONY: major
major: ## Increment the major version (X.y.z -> (X+1).0.0).
	@NEW_VERSION=$(shell echo $(VERSION) | awk -F. '{print $$1+1".0.0"}') && \
	$(SED) -i -E "s/(const version string = \"v)[^\"]+/\1$${NEW_VERSION}/" cmd/main.go && \
	$(MAKE) update-version VERSION=$${NEW_VERSION}

.PHONY: update-version
update-version: ## Update deployment manifests with the new version.
	@echo "Updating version to $(VERSION)"
	@find deploy/kubernetes -type f -name '*.yaml' -exec $(SED) -i -E "s/(image:\s*.*cascader:).*/\1v$(VERSION)/" {} \;
	@$(SED) -i -E "s/(appVersion\:)\s.*/\1 v$(VERSION)/" deploy/kubernetes/chart/cascader/Chart.yaml

.PHONY: tag
tag: ## Tag the current commit with the current version if no tag exists and the repository is clean.
	@if [ -n "$(shell git status --porcelain)" ]; then \
		echo "Repository has uncommitted changes. Please commit or stash them before tagging."; \
		exit 1; \
	fi
	@if [ -z "$(shell git tag --list v$(VERSION))" ]; then \
		echo "Tagging version v$(VERSION)"; \
		git tag v$(VERSION); \
		git push origin v$(VERSION); \
	else \
		echo "Tag v$(VERSION) already exists."; \
	fi

##@ Development

.PHONY: download
download: ## Download go packages
	go mod download

.PHONY: update-packages
update-packages: ## Update all Go packages to their latest versions
	go get -u ./...
	go mod tidy

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet envtest ## Run unit tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test $$(go list ./... | grep -v /e2e) -coverprofile cover.out

.PHONY: kind
kind: $(KIND) ## Create a Kind cluster.
	@echo "Setting up Kind cluster..."
	@$(KIND) create cluster --name cascader-test --wait 60s
	@kubectl cluster-info

.PHONY: delete-kind
delete-kind: ## Delete the Kind cluster.
	@echo "Deleting Kind cluster..."
	@$(KIND) delete cluster --name cascader-test
	@echo "Kind cluster teardown complete."

.PHONY: e2e
e2e: ## Run the e2e tests against an existing Kubernetes cluster.
	@echo "Running e2e tests..."
	USE_EXISTING_CLUSTER="true" go test ./test/e2e/ -timeout=15m -v -ginkgo.v

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter.
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes.
	$(GOLANGCI_LINT) run --fix

.PHONY: check-header
check-header: ## Verify that all *.go files have the boilerplate header
	@missing_files=0; \
	for file in $(shell find . -type f -name '*.go'); do \
		if ! diff <(head -n $(shell wc -l < hack/boilerplate.go.txt) $$file) hack/boilerplate.go.txt > /dev/null; then \
			echo "Missing or incorrect header in $$file"; \
			missing_files=$$((missing_files + 1)); \
		fi; \
	done; \
	if [ $$missing_files -ne 0 ]; then \
		echo "ERROR: Some files are missing the required boilerplate header."; \
		exit 1; \
	fi; \
	echo "All files have the correct boilerplate header."

.PHONY: check-header-fix
check-header-fix: ## Fix missing or incorrect headers in all *.go files
	@for file in $(shell find . -type f -name '*.go'); do \
		if ! diff <(head -n $(shell wc -l < hack/boilerplate.go.txt) $$file) hack/boilerplate.go.txt > /dev/null; then \
			echo "Fixing header in $$file"; \
			content=$$(cat $$file); \
			cat hack/boilerplate.go.txt > $$file; \
			echo "" >> $$file; \
			echo "$$content" >> $$file; \
		fi; \
	done; \
	echo "Headers have been fixed for all *.go files."

##@ Build

.PHONY: build
build: fmt vet ## Build manager binary.
	go build -o bin/cascader cmd/main.go

.PHONY: run
run: fmt vet ## Run a controller from your host.
	go run ./cmd/main.go $(ARGS)

.PHONY: kustomize
kustomize: ## Render kustomize manifests and save as a single file.
	kustomize build deploy/kubernetes/ > deploy/kubernetes/cascader.yaml
	yamlfmt deploy/kubernetes/kustomized.yaml

##@ Dependencies

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

.PHONY: yamlfmt
yamlfmt: $(LOCALBIN)/yamlfmt ## Download yamlfmt locally if necessary.
$(LOCALBIN)/yamlfmt: $(LOCALBIN)
	$(call go-install-tool,$(LOCALBIN)/yamlfmt,github.com/google/yamlfmt/cmd/yamlfmt,$(YAMLFMT_VERSION))

.PHONY: kind
$(KIND): $(LOCALBIN)
	@if [ ! -f $(KIND) ]; then \
		echo "Downloading Kind v$(KIND_VERSION) for $(UNAME)..."; \
		curl -L -o $(KIND) https://github.com/kubernetes-sigs/kind/releases/download/v$(KIND_VERSION)/$(KIND_BINARY); \
		chmod +x $(KIND); \
		echo "Kind v$(KIND_VERSION) installed at $(KIND)."; \
	fi

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef

