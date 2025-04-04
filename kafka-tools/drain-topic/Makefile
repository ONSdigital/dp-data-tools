SHELL=bash

# some vars are 'export'ed because ./scripts/ensure_vars needs them

export ENV?=sandbox
export SUBNET?=publishing

# APP is in dp-kafka/examples
APP?=consumer-batch

TOPIC?=observation-extracted
GROUP?=dp-observation-importer
export CERT_APP?=$(GROUP)

DP_CONFIGS?=../../../dp-configs
DP_KAFKA?=../../../dp-kafka
export DP_SETUP?=../../../dp-setup
export KEY_ADMIN_DIR?=$(DP_SETUP)/csr/private
LOG_QUIET?=1

host_num?=publishing 3
host_bin=bin-$(APP)

GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

BUILD=build
BUILD_ARCH=$(BUILD)/$(GOOS)-$(GOARCH)
BUILD_SCRIPT=$(BUILD_ARCH)/$(APP).sh

########################################

drain:
	GOOS=linux GOARCH=amd64 $(MAKE) clean-deploy deploy clean

pre-build: ensure-dirs

ensure-dirs:
	[[ -d $(DP_CONFIGS) ]]
	[[ -d $(DP_KAFKA) ]]
	mkdir -p $(BUILD_ARCH)

# convert secrets to env vars, add env vars for APP
env-vars:
	@$(DP_CONFIGS)/scripts/secrets-admin $(ENV) $(SUBNET) $(CERT_APP) --export 'KAFKA_*'
	@echo export SNOOZE=false KAFKA_PARALLEL_MESSAGES=12 KAFKA_BATCH_SIZE=100 LOG_QUIET=$(LOG_QUIET)
	@echo export KAFKA_CONSUMED_TOPIC="$(TOPIC)" KAFKA_CONSUMED_GROUP="$(GROUP)"

build-bin: pre-build
	cwd=$(PWD) && cd $(DP_KAFKA) && GOOS=$(GOOS) go build -o $$cwd/$(BUILD_ARCH)/$(APP) examples/$(APP)/main.go

build-script: pre-build
	$(MAKE) env-vars > $(BUILD_SCRIPT)

build: pre-build build-bin build-script

deploy: build clean-deploy
	dp scp $(ENV) $(host_num) -r -- $(BUILD_ARCH)/. $(host_bin)
	dp ssh $(ENV) $(host_num) -- 'bash -c "cd $(host_bin) && source ./$(APP).sh && ./$(APP)"'

clean: clean-deploy
	-rm -r $(BUILD)

clean-deploy:
	dp ssh $(ENV) $(host_num) -- 'bash -c "[[ ! -d $(host_bin) ]] || rm -r $(host_bin)"'

.PHONY: drain pre-build ensure-dirs env-vars build build-bin build-script deploy clean clean-deploy
