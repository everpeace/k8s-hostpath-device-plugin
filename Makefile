IMAGE ?= k8s-hostpath-device-plugin
.PHONY: build clean build-image dev-cluster dev-deply dev-clean

build:
	go build -o bin/k8s-hostpath-device-plugin .
clean:
	rm -rf bin/
build-image:
	docker build . -t $(IMAGE)

dev-cluster:
	helm repo add jetstack https://charts.jetstack.io
	helm repo update
	kind get kubeconfig >/dev/null 2>&1 || kind create cluster
	docker exec kind-control-plane sh -c 'mkdir -p /sample && echo "hello" > /sample/hello'
	mkdir -p tmp && kind get kubeconfig > tmp/kubeconfig && chmod 600 tmp/kubeconfig
	KUBECONFIG=tmp/kubeconfig helm install \
		cert-manager jetstack/cert-manager \
		--namespace cert-manager \
		--create-namespace \
		--version v1.5.3 \
		--set installCRDs=true

dev-deploy: build-image
	kind load docker-image $(IMAGE)
	if [ $(IMAGE) != "k8s-hostpath-device-plugin" ]; then \
		cd example; \
		kustomize edit set image k8s-hostpath-device-plugin $(IMAGE); \
	fi
	kustomize build example/ |  kubectl apply -f -
	KUBECONFIG=tmp/kubeconfig kubectl rollout status deployment \
		-n hostpath-sample-device-plugin \
		hostpath-sample-device-plugin-webhook \
		--timeout 60s
	KUBECONFIG=tmp/kubeconfig kubectl rollout status daemonset \
		-n hostpath-sample-device-plugin \
		hostpath-sample-device-plugin-ds \
		--timeout 60s

dev-clean:
	kind delete cluster && rm -rf tmp/

