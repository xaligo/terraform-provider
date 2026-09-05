COMPOSE ?= docker compose
DOCKER ?= docker
TERRAFORM ?= terraform
GO ?= go

-include .env

SAMPLE_DIR ?= samples/simple-vpc
GO_VERSION ?= 1.25.8
PROVIDER_VERSION ?= 0.1.0
PROVIDER_IMAGE ?= terraform-provider-xaligo:dev
AWS_DEFINITIONS_TOOL ?= ./tools/awsdefinitions
MINISTACK_PORT ?= 4566
MINISTACK_ENDPOINT ?= http://127.0.0.1:$(MINISTACK_PORT)

HOST_OS ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
HOST_MACHINE ?= $(shell uname -m)
HOST_ARCH ?= $(if $(filter x86_64,$(HOST_MACHINE)),amd64,$(if $(filter aarch64,$(HOST_MACHINE)),arm64,$(HOST_MACHINE)))

PROVIDER_MIRROR ?= $(CURDIR)/$(SAMPLE_DIR)/terraform.d/plugins
PROVIDER_DIR := $(PROVIDER_MIRROR)/registry.terraform.io/xaligo/xaligo/$(PROVIDER_VERSION)/$(HOST_OS)_$(HOST_ARCH)
PROVIDER_BINARY := $(PROVIDER_DIR)/terraform-provider-xaligo_v$(PROVIDER_VERSION)

.PHONY: ministack-up ministack-logs ministack-down provider-install provider-install-local provider-image provider-version convert tf-init tf-plan tf-apply tf-destroy sample aws-definitions-update aws-definitions-generate aws-definitions-check

ministack-up:
	$(COMPOSE) up -d --wait ministack

ministack-logs:
	$(COMPOSE) logs -f ministack

ministack-down:
	$(COMPOSE) down

# Build inside Docker, but export a native binary for the host Terraform CLI.
provider-install:
	mkdir -p "$(PROVIDER_DIR)"
	$(DOCKER) build \
		--target provider-artifact \
		--build-arg GO_VERSION="$(GO_VERSION)" \
		--build-arg PROVIDER_OS="$(HOST_OS)" \
		--build-arg PROVIDER_ARCH="$(HOST_ARCH)" \
		--build-arg PROVIDER_VERSION="$(PROVIDER_VERSION)" \
		--output "type=local,dest=$(PROVIDER_DIR)" \
		.
	chmod 0755 "$(PROVIDER_BINARY)"

# Optional fallback for contributors who already have the pinned Go toolchain.
provider-install-local:
	mkdir -p "$(PROVIDER_DIR)"
	CGO_ENABLED=0 GOOS="$(HOST_OS)" GOARCH="$(HOST_ARCH)" \
		$(GO) build -trimpath \
		-ldflags="-s -w -X main.version=$(PROVIDER_VERSION)" \
		-o "$(PROVIDER_BINARY)" .

provider-image:
	$(DOCKER) build \
		--target provider-image \
		--build-arg GO_VERSION="$(GO_VERSION)" \
		--build-arg PROVIDER_VERSION="$(PROVIDER_VERSION)" \
		--tag "$(PROVIDER_IMAGE)" \
		.

provider-version: provider-install
	"$(PROVIDER_BINARY)" version

aws-definitions-update:
	$(GO) run $(AWS_DEFINITIONS_TOOL) update

aws-definitions-generate:
	$(GO) run $(AWS_DEFINITIONS_TOOL) generate

aws-definitions-check:
	$(GO) run $(AWS_DEFINITIONS_TOOL) generate -check
	$(GO) run $(AWS_DEFINITIONS_TOOL) check-latest

convert: provider-install
	"$(PROVIDER_BINARY)" convert "$(SAMPLE_DIR)/source" \
		--output ../generated.xal \
		--title "Simple VPC"

tf-init: provider-install ministack-up
	env -C "$(SAMPLE_DIR)" TF_VAR_ministack_endpoint="$(MINISTACK_ENDPOINT)" \
		$(TERRAFORM) init -input=false

tf-plan: provider-install ministack-up
	env -C "$(SAMPLE_DIR)" TF_VAR_ministack_endpoint="$(MINISTACK_ENDPOINT)" \
		$(TERRAFORM) plan -input=false

tf-apply: provider-install ministack-up
	env -C "$(SAMPLE_DIR)" TF_VAR_ministack_endpoint="$(MINISTACK_ENDPOINT)" \
		$(TERRAFORM) apply -auto-approve -input=false

tf-destroy: provider-install ministack-up
	env -C "$(SAMPLE_DIR)" TF_VAR_ministack_endpoint="$(MINISTACK_ENDPOINT)" \
		$(TERRAFORM) destroy -auto-approve -input=false

sample: provider-install ministack-up
	env -C "$(SAMPLE_DIR)" TF_VAR_ministack_endpoint="$(MINISTACK_ENDPOINT)" \
		$(TERRAFORM) init -input=false
	env -C "$(SAMPLE_DIR)" TF_VAR_ministack_endpoint="$(MINISTACK_ENDPOINT)" \
		$(TERRAFORM) apply -auto-approve -input=false
	diff -u "$(SAMPLE_DIR)/expected.xal" "$(SAMPLE_DIR)/generated.xal"
