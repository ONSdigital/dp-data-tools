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

host_num?=publishing 3
host_bin=bin-$(APP)

GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

BUILD=build
BUILD_ARCH=$(BUILD)/$(GOOS)-$(GOARCH)
BUILD_SCRIPT=$(BUILD_ARCH)/$(APP).sh

export SECRETS_JSON=$(DP_CONFIGS)/secrets/$(ENV)/$(CERT_APP).json
CERT_DIR=$(KEY_ADMIN_DIR)/keys/$(ENV)/$(SUBNET)/$(CERT_APP)
CERT_FILE_PREFIX=$(CERT_APP).$(SUBNET).kafka.$(ENV).dp.ons.private
VARS_SH=$(CERT_DIR)/$(CERT_FILE_PREFIX).vars.sh
export ARN_FILE=$(CERT_DIR)/$(CERT_FILE_PREFIX).arn

########################################

drain:
	GOOS=linux make clean-deploy deploy clean

pre-build: ensure-dirs ensure-vars

ensure-dirs:
	[[ -d $(DP_CONFIGS) ]]
	[[ -d $(DP_KAFKA) ]]
	mkdir -p $(BUILD_ARCH)

# drain-topic needs KAFKA_SEC_CLIENT_CERT from either key-admin (VARS_SH) or secrets
ensure-vars:
	@[[ -f $(VARS_SH) ]] || ./scripts/ensure_vars

# convert secrets to env vars, add env vars for APP
env-vars:
	@if [[ -f $(VARS_SH) ]]; then \
		cat $(VARS_SH); \
	elif [[ -f $(SECRETS_JSON) ]]; then \
		jq -r ' . as $$o |  keys | .[] | "export " + . + "=" + ($$o[.] | tojson)' $(SECRETS_JSON) | grep '^export KAFKA_' | sed 's/\\n/\n/g'; \
	fi
	@echo export SNOOZE=false KAFKA_PARALLEL_MESSAGES=12 KAFKA_BATCH_SIZE=100 LOG_QUIET=1
	@echo export KAFKA_CONSUMED_TOPIC="$(TOPIC)" KAFKA_CONSUMED_GROUP="$(GROUP)"

build-bin: pre-build
	cwd=$(PWD) && cd $(DP_KAFKA) && GOOS=$(GOOS) go build -o $$cwd/$(BUILD_ARCH)/$(APP) examples/$(APP)/main.go

build-script: pre-build
	make env-vars > $(BUILD_SCRIPT)

build: pre-build build-bin build-script

deploy: build clean-deploy
	dp scp $(ENV) $(host_num) -r -- $(BUILD_ARCH)/. $(host_bin)
	dp ssh $(ENV) $(host_num) -- 'bash -c "cd $(host_bin) && source ./$(APP).sh && ./$(APP)"'

clean: clean-deploy
	-rm -r $(BUILD)

clean-deploy:
	dp ssh $(ENV) $(host_num) -- 'bash -c "[[ ! -d $(host_bin) ]] || rm -r $(host_bin)"'

.PHONY: drain pre-build ensure-dirs ensure-vars env-vars build build-bin build-script deploy clean clean-deploy
