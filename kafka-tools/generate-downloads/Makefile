# see main.go for all env var options/defaults

SHELL=bash

INSTANCE_ID?=8e4b7e60-5836-4da4-bcb8-479c71a4aafc
DATASET_ID?=uk-spending-on-cards
VERSION?=1

APP?=generate-downloads
ENV?=develop
host_num?=publishing 3

GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

BUILD=build
BUILD_ARCH=$(BUILD)/$(GOOS)-$(GOARCH)
BUILD_SCRIPT=$(BUILD_ARCH)/$(APP).sh

DP_CONFIGS?=../../../dp-configs
SECRETS_APP?=dp-dataset-api-publishing
SECRETS_JSON=$(DP_CONFIGS)/secrets/$(ENV)/$(SECRETS_APP).json

.PHONY: pre-build
pre-build:
	mkdir -p $(BUILD_ARCH)

.PHONY: build
build: pre-build
	go build -o $(BUILD_ARCH)/$(APP) main.go

.PHONY: env-vars
env-vars:
	@jq -r ' . as $$o |  keys | .[] | 'export ' + . + "=" + ($$o[.] | tojson)' $(SECRETS_JSON) | sed 's/\\n/\n/g'
	@echo export INSTANCE_ID="$(INSTANCE_ID)" DATASET_ID="$(DATASET_ID)" VERSION=$(VERSION)

.PHONY: deploy
deploy:
	make GOOS=linux build deploy-arch deploy-clean

.PHONY: deploy-arch
deploy-arch: build
	make env-vars > $(BUILD_SCRIPT)
	dp scp $(ENV) $(host_num) -r -- $(BUILD_ARCH) bin-$(APP)
	dp ssh $(ENV) $(host_num) -- 'bash -c "cd bin-$(APP) && source ./$(APP).sh && ./$(APP)"'

.PHONY: deploy-clean
deploy-clean:
	dp ssh $(ENV) $(host_num) -- rm -r bin-$(APP)
