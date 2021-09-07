# env
export GO111MODULE=on
export CGO_ENABLED=0

# project metadta
NAME         := k8s-hostpath-device-plugin
VERSION      ?= $(if $(RELEASE),$(shell git semv now),$(shell git semv patch -p))
REVISION     ?= $(shell git rev-parse --short HEAD)
IMAGE_PREFIX ?= ghcr.io/everpeace/
IMAGE_NAME   := $(NAME)
IMAGE_TAG    ?= $(if $(RELEASE),$(VERSION),$(VERSION)-$(REVISION))
LDFLAGS      := -ldflags="-s -w -X \"github.com/everpeace/k8s-hostpath-device-plugin/cmd.Version=$(VERSION)\" -X \"github.com/everpeace/k8s-hostpath-device-plugin/cmd.Revision=$(REVISION)\" -extldflags \"-static\""
OUTDIR       ?= ./dist

.DEFAULT_GOAL := build

.PHONY: setup
setup:
	go install golang.org/x/tools/cmd/goimports@latest && \
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.27.0 && \
	go install github.com/linyows/git-semv/cmd/git-semv@latest && \
	go install sigs.k8s.io/kind@latest && \
	curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/kustomize/v4.3.0/hack/install_kustomize.sh" | bash && \
	curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash

.PHONY: fmt
fmt:
	goimports -w cmd/ pkg/

.PHONY: lint
lint: fmt
	golangci-lint run --config .golangci.yml --deadline 30m

.PHONY: build
build: fmt lint
	go build -tags netgo -installsuffix netgo $(LDFLAGS) -o $(OUTDIR)/$(NAME) .

.PHONY: build-only
build-only: 
	go build -tags netgo -installsuffix netgo $(LDFLAGS) -o $(OUTDIR)/$(NAME) .

.PHONY: test
test: fmt lint
	go test  ./...

.PHONY: clean
clean:
	rm -rf "$(OUTDIR)"

.PHONY: build-image
build-image:
	docker build -t $(shell make -e docker-tag) \
		--build-arg RELEASE=$(RELEASE) \
		--build-arg VERSION=$(VERSION) \
		--build-arg REVISION=$(REVISION) \
		--target runtime .
	docker tag $(shell make -e docker-tag) $(IMAGE_PREFIX)$(IMAGE_NAME):$(VERSION)  # without revision
	docker tag $(shell make -e docker-tag) $(IMAGE_PREFIX)$(IMAGE_NAME):latest      # latest

.PHONY: push-image
push-image:
	docker push $(shell make -e docker-tag)
	docker push $(IMAGE_PREFIX)$(IMAGE_NAME):$(VERSION) # without revision
	docker push $(IMAGE_PREFIX)$(IMAGE_NAME):latest     # latest

.PHONY: docker-tag
docker-tag:
	@echo $(IMAGE_PREFIX)$(IMAGE_NAME):$(IMAGE_TAG)

#
# Release
#
guard-%:
	@ if [ "${${*}}" = "" ]; then \
    echo "Environment variable $* is not set"; \
		exit 1; \
	fi
.PHONY: release
release: guard-RELEASE guard-RELEASE_TAG
	git diff --quiet HEAD || (echo "your current branch is dirty" && exit 1)
	git tag $(RELEASE_TAG) $(REVISION)
	git push origin $(RELEASE_TAG)

#
# Dev
#
DEV_DIR = .dev
DEV_KUBECONFIG = $(DEV_DIR)/kubeconfig
.PHONY: dev-cluster
dev-cluster:
	helm repo add jetstack https://charts.jetstack.io
	helm repo update
	kind get kubeconfig >/dev/null 2>&1 || kind create cluster
	docker exec kind-control-plane sh -c 'mkdir -p /sample && echo "hello" > /sample/hello'
	mkdir -p $(DEV_DIR) && kind get kubeconfig > $(DEV_KUBECONFIG) && chmod 600 $(DEV_KUBECONFIG)
	KUBECONFIG=$(DEV_KUBECONFIG) helm status cert-manager --namespace=cert-manager >/dev/null 2>&1 || \
	KUBECONFIG=$(DEV_KUBECONFIG) helm install \
		cert-manager jetstack/cert-manager \
		--namespace cert-manager \
		--create-namespace \
		--version v1.5.3 \
		--set installCRDs=true \
		--wait

.PHONY: dev-deploy
dev-deploy: build-image
	kind load docker-image $(shell make -e docker-tag)
	cd example && kustomize edit set image k8s-hostpath-device-plugin=$(shell make -e docker-tag)
	kustomize build example/ |  kubectl apply -f -
	KUBECONFIG=$(DEV_KUBECONFIG) kubectl rollout status deployment \
		-n hostpath-sample-device-plugin \
		hostpath-sample-device-plugin-webhook \
		--timeout 60s
	KUBECONFIG=$(DEV_KUBECONFIG) kubectl rollout status daemonset \
		-n hostpath-sample-device-plugin \
		hostpath-sample-device-plugin-ds \
		--timeout 60s

.PHONY: e2e
e2e: dev-cluster dev-deploy
	go test ./e2e --kubeconfig=../.dev/kubeconfig --pluginconfig=../example/config.yaml

.PHONY: dev-clean
dev-clean:
	kind delete cluster && rm -rf $(DEV_DIR)
