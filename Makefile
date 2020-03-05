GO    	 := go
pkgs      = $(shell $(GO) list ./... | grep -v /vendor/)
arch      = amd64  ## default architecture
platforms = darwin linux windows
package   = alertmanager-maintenance-scheduler

PREFIX                  ?= $(shell pwd)
BIN_DIR                 ?= $(shell pwd)
DOCKER_REPO             ?= fxinnovation
DOCKER_IMAGE_NAME       ?= $(package)
DOCKER_IMAGE_TAG        ?= $(subst /,-,$(shell git rev-parse --abbrev-ref HEAD))

all: vet format test build

build: ## build executable for current platform
	@echo ">> building..."
	@$(GO) build

xbuild: ## cross build executables for all defined platforms
	@echo ">> cross building executable(s)..."

	@for platform in $(platforms); do \
		echo "build for $$platform/$(arch)" ;\
		name=$(package)'-'$$platform'-'$(arch) ;\
		if [ $$platform = "windows" ]; then \
			name=$$name'.exe' ;\
		fi ;\
		echo $$name ;\
		GOOS=$$platform GOARCH=$(arch) $(GO) build -o $$name . ;\
	done

docker:
	@echo ">> building docker image"
	@docker build -t "$(DOCKER_REPO)/$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)" .

test:
	@echo ">> running tests.."
	@$(GO) test -v -short $(pkgs)

format: ## format code
	@echo ">> formatting code"
	@$(GO) fmt $(pkgs)

vet: ## vet code
	@echo ">> vetting code"
	@$(GO) vet $(pkgs)

lint: golint ## lint code
	@echo ">> linting code"
	@! golint $(pkgs) | grep '^'

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

golint: ## downloads golint
	@go get -u golang.org/x/lint/golint