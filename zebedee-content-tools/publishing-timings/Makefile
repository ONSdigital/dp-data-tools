APP=publish-timings

ENV?=sandbox

GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

BUILD=build
BUILD_ARCH=$(BUILD)/$(GOOS)-$(GOARCH)

.PHONY: build
build:
	go build -o $(BUILD_ARCH)/$(APP)

.PHONY: build-aws
build-aws:
	make CGO_ENABLED=0 GOOS=linux GOARCH=amd64 build

.PHONY: deploy
deploy: build-aws
	dp scp $(ENV) publishing_mount 1  -- $(BUILD)/linux-amd64/$(APP) $(APP);

.PHONY: clean
clean:
	[[ ! -d $(BUILD) ]] || rm -r $(BUILD)

