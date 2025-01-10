# env
export GO111MODULE=on
export CGO_ENABLED=0

# project metadta
NAME         := k8s-hostpath-device-plugin
REVISION     := $(shell git rev-parse --short HEAD)
IMAGE_PREFIX ?= ghcr.io/everpeace/
IMAGE_NAME   := $(NAME)
IMAGE_TAG    ?= dev-$(REVISION)
LDFLAGS      := -ldflags="-s -w -X github.com/everpeace/k8s-hostpath-device-plugin/cmd.Revision=$(REVISION) -extldflags \"-static\""
OUTDIR       ?= ./dist

.DEFAULT_GOAL := build

.PHONY: fmt
fmt: goimports
	$(GOIMPORTS) -w cmd/ pkg/

.PHONY: lint
lint: fmt golangci-lint
	$(GOLANGCI_LINT) run --config .golangci.yml --timeout 30m

.PHONY: build
build: fmt lint
	go build -tags netgo -installsuffix netgo $(LDFLAGS) -o $(OUTDIR)/$(NAME) .

.PHONY: build-only
build-only: 
	go build -tags netgo -installsuffix netgo $(LDFLAGS) -o $(OUTDIR)/$(NAME) .

.PHONY: test
test: fmt lint
	go test  ./cmd/... ./pkg/...

.PHONY: e2e-test
e2e-test: dev-cluster dev-deploy
	go test ./e2e --kubeconfig=../.dev/kubeconfig --pluginconfig=../example/config.yaml

.PHONY: clean
clean:
	rm -rf "$(OUTDIR)"

.PHONY: build-image
build-image:
	docker build -t $(shell make -e docker-tag) \
		--target runtime .

.PHONY: docker-tag
docker-tag:
	@echo $(IMAGE_PREFIX)$(IMAGE_NAME):$(IMAGE_TAG)

#
# Dev
#
DEV_DIR = .dev
DEV_KUBECONFIG = $(DEV_DIR)/kubeconfig
KIND_NODE_IMAGE ?= kindest/node:v1.31.4
.PHONY: dev-cluster
dev-cluster: kind helm
	$(HELM) repo add jetstack https://charts.jetstack.io
	$(HELM) repo update
	$(KIND) get kubeconfig >/dev/null 2>&1 || $(KIND) create cluster --image=$(KIND_NODE_IMAGE)
	docker exec kind-control-plane sh -c 'mkdir -p /sample && echo "hello" > /sample/hello'
	mkdir -p $(DEV_DIR) && $(KIND) get kubeconfig > $(DEV_KUBECONFIG) && chmod 600 $(DEV_KUBECONFIG)
	KUBECONFIG=$(DEV_KUBECONFIG) $(HELM) status cert-manager --namespace=cert-manager >/dev/null 2>&1 || \
	KUBECONFIG=$(DEV_KUBECONFIG) $(HELM) install \
		cert-manager jetstack/cert-manager \
		--namespace cert-manager \
		--create-namespace \
		--version v1.15.3 \
		--set installCRDs=true \
		--wait

.PHONY: dev-deploy
dev-deploy: build-image kind kustomize
	$(KIND) load docker-image $(shell make -e docker-tag)
	cd example && $(KUSTOMIZE) edit set image k8s-hostpath-device-plugin=$(shell make -e docker-tag)
	$(KUSTOMIZE) build example/ |  kubectl apply -f -
	KUBECONFIG=$(DEV_KUBECONFIG) kubectl rollout status deployment \
		-n hostpath-sample-device-plugin \
		hostpath-sample-device-plugin-webhook \
		--timeout 60s
	KUBECONFIG=$(DEV_KUBECONFIG) kubectl rollout status daemonset \
		-n hostpath-sample-device-plugin \
		hostpath-sample-device-plugin-ds \
		--timeout 60s

.PHONY: dev-clean
dev-clean: kind
	$(KIND) delete cluster && rm -rf $(DEV_DIR)


#
# Dev Dependencies
#
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KIND ?= $(LOCALBIN)/kind
KUSTOMIZE ?= $(LOCALBIN)/kustomize
HELM ?= $(LOCALBIN)/helm
GOIMPORTS ?= $(LOCALBIN)/goimports
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

## Tool Versions
KUSTOMIZE_VERSION ?= v5.4.3
GOLANGCI_LINT_VERSION ?= v1.61.0

.PHONY: goimports
goimports: $(GOIMPORTS) ## Download goimports locally if necessary.
$(GOIMPORTS): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install golang.org/x/tools/cmd/goimports@latest

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
$(KUSTOMIZE): $(LOCALBIN)
	curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN)

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
GOLANGCI_LINT_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh"
$(GOLANGCI_LINT): $(LOCALBIN)
	curl -sSfL $(GOLANGCI_LINT_INSTALL_SCRIPT) | sh -s -- -b $(LOCALBIN) $(GOLANGCI_LINT_VERSION)

.PHONY: kind
kind: $(KIND) ## Download kind locally if necessary.
$(KIND): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/kind@latest

.PHONY: helm
helm: $(HELM) ## Download helm locally if necessary.
HELM_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3"
$(HELM): $(LOCALBIN)
	curl -s $(HELM_INSTALL_SCRIPT) | USE_SUDO=false HELM_INSTALL_DIR=$(LOCALBIN) bash
