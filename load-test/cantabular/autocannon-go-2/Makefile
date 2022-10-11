# see autocannon.go for all env var options/defaults

SHELL=bash
APP=generate-load

ENV?=staging

SUBNET?=publishing

# adjust the number of host_numbers to suite the desired load for your test
host_numbers = 3 4 5 6
#7 8 9 10

host_bin=bin-$(APP)

GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

BUILD=build
BUILD_ARCH=$(BUILD)/$(GOOS)-$(GOARCH)
BUILD_SCRIPT=$(BUILD_ARCH)/$(APP).sh

.PHONY: all
all: clean
# We clean (above line) first to start with fresh and known starting point and after
# the deploy has run we can (if needs be) inspect any log files on the boxes
	make GOOS=linux build deploy

.PHONY: build
build:
	go build -o $(BUILD_ARCH)/$(APP) autocannon.go

.PHONY: script
script:
	cp $(APP).sh $(BUILD_SCRIPT)
	chmod +x $(BUILD_SCRIPT)

.PHONY: env-vars
env-vars:
	@echo '#'
	@echo ./$(APP)

.PHONY: deploy
deploy: build script
# first copy application and its launch script on to all hosts
	for num in $(host_numbers); do\
		dp scp $(ENV) $(SUBNET) $$num -r -- $(BUILD_ARCH) $(host_bin); \
	done

# then setup the application on each host to run 'n' minutes in the future, where 'n' is
# a number big enough to allow the following for loop to complete.
# ('n' has to be determined by observation).
	TZ=UTC; start_time=$(shell TZ=UTC date -v +3M '+%H:%M'); \
	for num in $(host_numbers); do\
		dp ssh $(ENV) $(SUBNET) $$num -- 'bash -c "cd $(host_bin) && ./$(APP).sh '"$$start_time"'"' ; \
	done

.PHONY: launch
launch:
# run the load test apps again (that have already been build and deployed)
	TZ=UTC; start_time=$(shell TZ=UTC date -v +3M '+%H:%M'); \
	for num in $(host_numbers); do\
		dp ssh $(ENV) $(SUBNET) $$num -- 'bash -c "cd $(host_bin) && ./$(APP).sh '"$$start_time"'"' ; \
	done


.PHONY: clean
clean: clean-deploy clean-build

.PHONY: clean-build
clean-build:
	[[ ! -d $(BUILD) ]] || rm -r $(BUILD)

.PHONY: clean-deploy
clean-deploy:
	for num in $(host_numbers); do\
		dp ssh $(ENV) $(SUBNET) $$num -- bash -c '"[[ ! -d $(host_bin) ]] || rm -r $(host_bin)"'; \
	done

.PHONY: clean-tmp
clean-tmp:
	for num in $(host_numbers); do\
		dp ssh $(ENV) $(SUBNET) $$num -- bash -c '"[[ ! -d tmp ]] || rm -r tmp"'; \
	done
