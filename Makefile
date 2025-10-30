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
GOLANGCI_LINT_VERSION ?= v2.6.0
# renovate: datasource=github-releases depName=google/yamlfmt
YAMLFMT_VERSION ?= v0.20.0
# renovate: datasource=github-releases depName=kubernetes-sigs/kind
KIND_VERSION ?= 0.30.0
# renovate: datasource=github-releases depName=onsi/ginkgo
GINKGO_VERSION ?= v2.27.2

.PHONY: all
all: build

##@ General
.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Tagging

# Find the latest tag (with prefix filter if defined, default to 0.0.0 if none found)
# Lazy evaluation ensures fresh values on every run
VERSION_PREFIX ?= v
LATEST_TAG = $(shell git tag --list "$(VERSION_PREFIX)*" --sort=-v:refname | head -n 1)
VERSION = $(shell [ -n "$(LATEST_TAG)" ] && echo $(LATEST_TAG) | sed "s/^$(VERSION_PREFIX)//" || echo "0.0.0")

patch: ## Create a new patch release (x.y.Z+1)
	@NEW_VERSION=$$(echo "$(VERSION)" | awk -F. '{printf "%d.%d.%d", $$1, $$2, $$3+1}') && \
	$(MAKE) update-version VERSION=$${NEW_VERSION} && \
	echo "Tagged $(VERSION_PREFIX)$${NEW_VERSION}"

minor: ## Create a new minor release (x.Y+1.0)
	@NEW_VERSION=$$(echo "$(VERSION)" | awk -F. '{printf "%d.%d.0", $$1, $$2+1}') && \
	$(MAKE) update-version VERSION=$${NEW_VERSION} && \
	echo "Tagged $(VERSION_PREFIX)$${NEW_VERSION}"

major: ## Create a new major release (X+1.0.0)
	@NEW_VERSION=$$(echo "$(VERSION)" | awk -F. '{printf "%d.0.0", $$1+1}') && \
	$(MAKE) update-version VERSION=$${NEW_VERSION} && \
	echo "Tagged $(VERSION_PREFIX)$${NEW_VERSION}"

tag: ## Show latest tag
	@echo "Latest version: $(LATEST_TAG)"

push: ## Push tags to remote
	git push --tags

.PHONY: update-version
update-version: ## Update deployment manifests with the new version.
	@find deploy/kubernetes -type f -name '*.yaml' -exec $(SED) -i -E "s/(image:\s*.*cascader:).*/\1v$(VERSION)/" {} \;
	@$(SED) -i -E "s/(appVersion\:)\s.*/\1 v$(VERSION)/" deploy/kubernetes/chart/cascader/Chart.yaml

##@ Development

.PHONY: download
download: ## Download go packages and list them.
	go mod download
	go list -m all

.PHONY: verify-deps
verify-deps: ## Verify go.mod and go.sum are tidy
	go mod tidy
	git diff --exit-code go.mod go.sum

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet envtest ## Run unit tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
	go test -coverprofile=cover.out -covermode=atomic -count=1 -parallel=4 -timeout=5m ./internal/...


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
e2e: ginkgo ## Run all e2e tests sequentially (Ginkgo procs=1 required due to shared state: LogBuffer, Operator process, Cluster resources)
	@echo "Running e2e tests with Ginkgo..."
	PATH=$(LOCALBIN):$$PATH USE_EXISTING_CLUSTER="true" \
  ginkgo --procs=1 --timeout=30m -v --focus='${FOCUS}' ./test/e2e/...

.PHONY: e2e-deployment
e2e-deployment: ## Run only Deployment e2e tests
	@$(MAKE) e2e FOCUS=Deployment

.PHONY: e2e-statefulset
e2e-statefulset: ## Run only StatefulSet e2e tests
	@$(MAKE) e2e FOCUS=StatefulSet

.PHONY: e2e-daemonset
e2e-daemonset: ## Run only DaemonSet e2e tests
	@$(MAKE) e2e FOCUS=DaemonSet

.PHONY: e2e-mixed
e2e-mixed: ## Run only Mixed workload e2e tests
	@$(MAKE) e2e FOCUS=Mixed

.PHONY: e2e-cycle
e2e-cycle: ## Run only Cycle detection e2e tests
	@$(MAKE) e2e FOCUS=Cycle

.PHONY: e2e-namespace
e2e-namespace: ## Run only Namespace watching e2e tests
	@$(MAKE) e2e FOCUS=Namespace

.PHONY: e2e-edgecases
e2e-edgecases: ## Run only Edge Case e2e tests
	@$(MAKE) e2e FOCUS=Edge

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
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

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

.PHONY: ginkgo
ginkgo: $(LOCALBIN)/ginkgo ## Download ginkgo locally if necessary.
$(LOCALBIN)/ginkgo: $(LOCALBIN)
	$(call go-install-tool,$(LOCALBIN)/ginkgo,github.com/onsi/ginkgo/v2/ginkgo,$(GINKGO_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
tools: ginkgo envtest golangci-lint yamlfmt kind ## Install all tools

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

