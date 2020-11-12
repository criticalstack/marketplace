NAME := manager

IMG_TAG ?= latest
IMG ?= docker.io/criticalstack/marketplace:$(IMG_TAG)

KUBEBUILDER_VERSION ?= 2.3.1

BIN_DIR := bin
RELEASE_DIR := dist

TOOLS_DIR      := $(shell pwd)/hack/tools
TOOLS_BIN_DIR  := $(TOOLS_DIR)/bin
CONTROLLER_GEN := $(TOOLS_BIN_DIR)/controller-gen

GO111MODULE := on
CGO_ENABLED := 0

API_DIR := api
API_SRC := $(shell find $(API_DIR) -type f \( -name '*.go' -not -name 'zz_generated.*' \))
API_DEEPCOPY := $(addsuffix /zz_generated.deepcopy.go,$(shell find $(API_DIR) -type d -not -name $(API_DIR)))

OBJECT_HEADER := hack/boilerplate.go.txt
CONTROLLER_GEN_CRD_OPTIONS ?= crd:trivialVersions=true
CONTROLLER_GEN_OBJECT_OPTIONS ?= object:headerFile=$(OBJECT_HEADER)

export KUBEBUILDER_ASSETS := $(TOOLS_BIN_DIR)
KUBEBUILDER_ASSETS_BIN := $(addprefix $(TOOLS_BIN_DIR)/,kubebuilder kube-apiserver etcd kubectl)

##@ Build

.PHONY: build run test fmt vet image

build: $(BIN_DIR)/$(NAME) ## Build controller-manager binary

$(BIN_DIR)/$(NAME): deepcopy-gen fmt
	go build -o $@ main.go

run: deepcopy-gen fmt ## Run against the configured Kubernetes cluster in ~/.kube/config
	go run ./main.go

test: $(KUBEBUILDER_ASSETS_BIN) deepcopy-gen vet ## Run go tests
	go test ./... -coverprofile cover.out

fmt: ## Run go fmt against code
	go fmt ./...

vet: ## Run go vet against code
	go vet ./...

image: ## Build manager container image
	docker build . -t ${IMG}

##@ Codegen

.PHONY: deepcopy-gen crds

deepcopy-gen: $(API_DEEPCOPY) ## Generate k8s object deepcopy methods

$(API_DEEPCOPY): $(CONTROLLER_GEN) $(OBJECT_HEADER) $(API_SRC)
	$(CONTROLLER_GEN) $(CONTROLLER_GEN_OBJECT_OPTIONS) paths="./..."

crds: manifests/crds ## Generate CRD manifests

manifests/crds: $(CONTROLLER_GEN) $(API_SRC)
	$(CONTROLLER_GEN) $(CONTROLLER_GEN_CRD_OPTIONS) paths="./..." output:crd:artifacts:config=$@
	@touch $@ # touch the directory to update the timestamp in case no new files were created

##@ Deploy

.PHONY: deploy install uninstall

deploy: crds ## Deploy CRDs + controller in the configured Kubernetes cluster in ~/.kube/configs
	kubectl apply -f manifests/

install: crds ## Install CRDs into a cluster
	kubectl apply -f manifests/crds

uninstall: crds ## Uninstall CRDs from a cluster
	kubectl delete -f manifests/crds

##@ Helpers

.PHONY: release clean help

release: $(RELEASE_DIR)/marketplace.yaml

$(RELEASE_DIR):
	mkdir -p $(RELEASE_DIR)/

YAMLS := $(shell find manifests/ -name '*yaml')

$(RELEASE_DIR)/marketplace.yaml: $(RELEASE_DIR) manifests/crds $(YAMLS) ## Set $IMG and join manifests into dist/marketplace.yaml
	cat $(YAMLS) | sed -e 's@image: .*@image: '"${IMG}"'@' > $@

$(KUBEBUILDER_ASSETS_BIN):
	mkdir -p $(KUBEBUILDER_ASSETS)
	curl -L https://go.kubebuilder.io/dl/$(KUBEBUILDER_VERSION)/$(shell go env GOOS)/$(shell go env GOARCH) | tar -xz -C /tmp/
	mv /tmp/kubebuilder_$(KUBEBUILDER_VERSION)_$(shell go env GOOS)_$(shell go env GOARCH)/bin/* $(KUBEBUILDER_ASSETS)/

$(CONTROLLER_GEN): $(TOOLS_DIR)/go.mod # Build controller-gen from tools folder.
	cd $(TOOLS_DIR); go build -o bin/controller-gen sigs.k8s.io/controller-tools/cmd/controller-gen

clean: ## Cleanup build folders
	rm -f $(BIN_DIR)/*
	rm -f $(TOOLS_BIN_DIR)/*

help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
