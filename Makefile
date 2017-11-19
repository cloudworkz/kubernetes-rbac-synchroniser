GO := GO15VENDOREXPERIMENT=1 go
pkgs = $(shell $(GO) list ./... | grep -v /vendor/)
DOCKER_IMAGE_NAME ?= yacut/kubernetes-rbac-synchroniser
DOCKER_IMAGE_TAG ?= $(subst /,-,$(shell git rev-parse --abbrev-ref HEAD))

test:
	@echo ">> running tests"
	@$(GO) test -short $(pkgs)

format:
	@echo ">> formatting code"
	@$(GO) fmt $(pkgs)

build:
	@echo ">> building binaries"
	@$(GO) build -o bin/kubernetes-rbac-synchroniser

docker:
	@echo ">> building docker image"
	@docker build -t "$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)" .
