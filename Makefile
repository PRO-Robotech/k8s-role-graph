SHELL := /bin/bash
GOCACHE := $(CURDIR)/.cache/go-build
GOMODCACHE := $(CURDIR)/.cache/go-mod

GOLANGCI_LINT_VERSION := v2.3.0

.PHONY: fmt lint test generate build-apiserver build-web docker-apiserver docker-web kustomize-kind openapi-spec verify-openapi-spec

generate:
	./hack/update-codegen.sh

fmt:
	gofmt -w $$(find cmd internal pkg -name '*.go')

lint:
	@which golangci-lint > /dev/null 2>&1 || \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	golangci-lint run ./...

test:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go test ./...

build-apiserver:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build -o bin/rbacgraph-apiserver ./cmd/rbacgraph-apiserver

build-web:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build -o bin/rbacgraph-web ./cmd/rbacgraph-web

docker-apiserver:
	docker build -f Dockerfile.apiserver -t rbacgraph-apiserver:dev .

docker-web:
	docker build -f Dockerfile.web -t rbacgraph-web:dev .

kustomize-kind:
	kubectl kustomize deploy/kustomize/overlays/kind

openapi-spec:
	go run ./hack/openapi-spec > api/openapi-spec/swagger.json

verify-openapi-spec:
	@diff <(go run ./hack/openapi-spec) api/openapi-spec/swagger.json || \
	  (echo "api/openapi-spec/swagger.json is stale — run 'make openapi-spec'" && exit 1)
