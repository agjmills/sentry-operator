# sentry-operator Makefile

BINARY      ?= manager
IMG         ?= ghcr.io/agjmills/sentry-operator:latest
PLATFORM    ?= linux/amd64

# Tool versions
CONTROLLER_GEN_VERSION ?= v0.16.0
ENVTEST_VERSION         ?= release-0.19

## Build

.PHONY: build
build: ## Build the manager binary
	go build -o bin/$(BINARY) ./cmd/main.go

.PHONY: run
run: ## Run the controller locally (uses current kubeconfig)
	go run ./cmd/main.go \
		--sentry-url=$(SENTRY_URL) \
		--default-organization=$(SENTRY_ORG) \
		--default-team=$(SENTRY_TEAM)

.PHONY: test
test: envtest ## Run unit and integration tests
	KUBEBUILDER_ASSETS="$(shell $(LOCALBIN)/setup-envtest use $(ENVTEST_VERSION) --bin-dir $(LOCALBIN) -p path)" \
		go test -race ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run ./...

## Code generation

.PHONY: generate
generate: controller-gen ## Regenerate zz_generated.deepcopy.go
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: manifests
manifests: controller-gen ## Regenerate CRD manifests under config/crd/bases/
	$(CONTROLLER_GEN) crd paths="./..." output:crd:artifacts:config=config/crd/bases
	cp config/crd/bases/sentry-operator.io_sentryprojects.yaml \
	   charts/sentry-operator/crds/sentry-operator.io_sentryprojects.yaml

## Docker

.PHONY: docker-build
docker-build: ## Build Docker image
	docker buildx build --platform=$(PLATFORM) -t $(IMG) .

.PHONY: docker-push
docker-push: ## Push Docker image
	docker push $(IMG)

## Deployment

.PHONY: install
install: manifests ## Install CRDs into the cluster
	kubectl apply -f config/crd/bases/

.PHONY: uninstall
uninstall: ## Remove CRDs from the cluster
	kubectl delete -f config/crd/bases/ --ignore-not-found

.PHONY: deploy
deploy: ## Deploy the operator to the cluster (for development)
	kubectl apply -f config/crd/bases/
	helm upgrade --install sentry-operator charts/sentry-operator/ \
		--set operator.defaultOrganization=$(SENTRY_ORG) \
		--set operator.defaultTeam=$(SENTRY_TEAM) \
		--set sentryToken=$(SENTRY_TOKEN)

## Tools

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
.PHONY: controller-gen
controller-gen: $(LOCALBIN)
	@test -s $(CONTROLLER_GEN) || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION)

ENVTEST ?= $(LOCALBIN)/setup-envtest
.PHONY: envtest
envtest: $(LOCALBIN)
	@test -s $(ENVTEST) || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: install-hooks
install-hooks: ## Configure git to use .githooks/
	git config core.hooksPath .githooks

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
