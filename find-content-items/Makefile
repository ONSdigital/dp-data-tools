BIN_NAME := find_content_items
ENV ?= sandbox 

.PHONY: all
all: build

.PHONY: build
build:
	GOOS=linux GOARCH=amd64 go build -o $(BIN_NAME) 'github.com/ONSdigital/dp-data-tools/find-content-items'

.PHONY: deploy
deploy: build
	@echo "Deploying to $(ENV)"
	@for SUBNET in web publishing; do \
		for HOST_NUM in 1 2; do \
			if [[ $$SUBNET == publishing && $$HOST_NUM -gt 1 ]]; then \
				continue; \
			fi; \
			dp scp $(ENV) $$SUBNET $$HOST_NUM $(BIN_NAME) . ; \
		done; \
	done