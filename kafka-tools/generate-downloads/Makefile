# see main.go for all env var options/defaults

SHELL=bash
APP=generate-downloads

ENV?=sandbox
INSTANCE_ID?=8e4b7e60-1136-4da4-bcb8-479c71a4aafc
DATASET_ID?=uk-spending-on-cards
EDITION?=time-series
VERSION?=1

TIMEOUT?=30s

host_num?=publishing 3
host_bin=bin-$(APP)

GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

BUILD=build
BUILD_ARCH=$(BUILD)/$(GOOS)-$(GOARCH)
BUILD_SCRIPT=$(BUILD_ARCH)/$(APP).sh

DP_CONFIGS?=../../../dp-configs

SERVICE?=cantabular

ifeq ($(SERVICE), cantabular)
	SECRETS_APP?=dp-cantabular-filter-flex-api-publishing
	KAFKA_PRODUCER_TOPIC?=cantabular-export-start
else
	SECRETS_APP?=dp-dataset-api-publishing
	KAFKA_PRODUCER_TOPIC?=filter-job-submitted
endif

SECRETS_JSON=$(DP_CONFIGS)/secrets/$(ENV)/$(SECRETS_APP).json

.PHONY: all
all: clean
	make GOOS=linux build deploy clean

.PHONY: pre-build
pre-build: secrets
	mkdir -p $(BUILD_ARCH)

.PHONY: build
build: pre-build
	go build -o $(BUILD_ARCH)/$(APP) main.go

.PHONY: script
script: pre-build
	make env-vars > $(BUILD_SCRIPT)

.PHONY: run
run: script
	. $(BUILD_SCRIPT); go run -race main.go

.PHONY: secrets
secrets:
	[[ -f $(SECRETS_JSON) ]] || gpg $(SECRETS_JSON).asc

.PHONY: env-vars
env-vars:
	@set -o pipefail; jq -r ' . as $$o |  keys | .[] | "export " + . + "=" + ($$o[.] | tojson)' $(SECRETS_JSON) | sed 's/\\n/\n/g'
	@echo export INSTANCE_ID="$(INSTANCE_ID)" DATASET_ID="$(DATASET_ID)"
	@echo export VERSION=$(VERSION) EDITION="$(EDITION)" TIMEOUT=$(TIMEOUT)
	@echo export KAFA_PRODUCER_TOPIC="$(KAFKA_PRODUCER_TOPIC)" SERVICE="$(SERVICE)"

.PHONY: deploy
deploy: build script
	dp scp $(ENV) $(host_num) -r -- $(BUILD_ARCH) $(host_bin)
	dp ssh $(ENV) $(host_num) -- 'bash -c "cd $(host_bin) && source ./$(APP).sh && ./$(APP)"'

.PHONY: clean
clean: clean-deploy clean-build

.PHONY: clean-build
clean-build:
	[[ ! -d $(BUILD) ]] || rm -r $(BUILD)

.PHONY: clean-deploy
clean-deploy:
	dp ssh $(ENV) $(host_num) -- bash -c '"[[ ! -d $(host_bin) ]] || rm -r $(host_bin)"'
