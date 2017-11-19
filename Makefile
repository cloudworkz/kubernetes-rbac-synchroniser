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

build:
	@echo ">> building binaries"
	@$(GO) build -o build/$(BINARY)

docker.build:
	@echo ">> building docker image"
	@docker build -t "$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)" -t "$(DOCKER_IMAGE_NAME):latest" .

build.push: docker.build
	@docker push "$(DOCKER_IMAGE_NAME)"

clean:
	@rm -rf build
