GO := GO15VENDOREXPERIMENT=1 go
BINARY ?= kubernetes-rbac-synchroniser
pkgs = $(shell $(GO) list ./... | grep -v /vendor/)
DOCKER_IMAGE_NAME ?= yacut/kubernetes-rbac-synchroniser
DOCKER_IMAGE_TAG ?= $(subst /,-,$(shell git describe --tags --always))

test:
	@echo ">> running tests"
	@$(GO) test -short $(pkgs)

format:
	@echo ">> formatting code"
	@$(GO) fmt $(pkgs)

install:
	@echo ">> installing dependencies"
	@go get -u k8s.io/client-go/...
	@go get -u github.com/prometheus/client_golang/...
	@go get -u golang.org/x/oauth2/...
	@go get -u google.golang.org/api/groupssettings/v1
	@go get -u google.golang.org/api/admin/directory/v1

build:
	@echo ">> building binaries"
	@$(GO) build -o build/$(BINARY)

docker.build:
	@echo ">> building docker image"
	@docker build -t "$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)" -t "$(DOCKER_IMAGE_NAME):latest" .

build.push:
	@docker push "$(DOCKER_IMAGE_NAME)"

clean:
	@rm -rf build
	@rm .credentials/kubernetes-rbac-synchroniser.json
