ENV?=develop

# APP is in dp-kafka/examples
APP?=consumer-batch

TOPIC?=observation-extracted
GROUP?=dp-observation-importer
CERT_APP?=$(GROUP)

DP_CONFIGS?=../../../dp-configs
DP_KAFKA?=../../../dp-kafka

host_num?=publishing 3
host_bin=bin-$(APP)

GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

BUILD=build
BUILD_ARCH=$(BUILD)/$(GOOS)-$(GOARCH)
BUILD_SCRIPT=$(BUILD_ARCH)/$(APP).sh

SECRETS_JSON=$(DP_CONFIGS)/secrets/$(ENV)/$(CERT_APP).json

########################################

drain:
	GOOS=linux make deploy clean

secrets:
	@[[ -f $(SECRETS_JSON) ]] || gpg -do $(SECRETS_JSON) $(SECRETS_JSON).asc

# convert secrets to env vars, add env vars for APP
env-vars: secrets
	@jq -r ' . as $$o |  keys | .[] | "export " + . + "=" + ($$o[.] | tojson)' $(SECRETS_JSON) | grep '^export KAFKA_' | sed 's/\\n/\n/g'
	@echo export SNOOZE=false KAFKA_PARALLEL_MESSAGES=12 KAFKA_BATCH_SIZE=100 LOG_QUIET=1
	@echo export KAFKA_CONSUMED_TOPIC="$(TOPIC)" KAFKA_CONSUMED_GROUP="$(GROUP)"

pre-build:
	[[ -d $(DP_CONFIGS) ]]
	[[ -d $(DP_KAFKA) ]]
	mkdir -p $(BUILD_ARCH)

build: pre-build
	cwd=$(PWD) && cd $(DP_KAFKA) && GOOS=$(GOOS) go build -o $$cwd/$(BUILD_ARCH)/$(APP) examples/$(APP)/main.go

script: secrets pre-build
	make env-vars > $(BUILD_SCRIPT)

deploy: build script clean-deploy
	dp scp $(ENV) $(host_num) -r -- $(BUILD_ARCH)/. $(host_bin)
	dp ssh $(ENV) $(host_num) -- 'bash -c "cd $(host_bin) && source ./$(APP).sh && ./$(APP)"'

clean: clean-deploy
	-rm -r $(BUILD)

clean-deploy:
	dp ssh $(ENV) $(host_num) -- 'bash -c "[[ ! -d $(host_bin) ]] || rm -r $(host_bin)"'

.PHONY: drain secrets env-vars pre-build build script deploy clean clean-deploy
