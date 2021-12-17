APP=check-audit
BUILD=build

ENV?=develop
HOST_NUM=publishing 3
CERT_D?=../../../dp-setup/csr/private

BIN=$(BUILD)/$(APP)
# following files sent to remote...
START_SH=run-$(APP).sh
START_PATH=$(BUILD)/$(START_SH)
# ...inside this bundle:
TARBALL=$(APP).tgz

# not export-ed, so that `test` works locally, and forces us to be explicit for `build` (POLS)
GOOS?=linux
GOARCH?=amd64

dir:
	@[[ -d $(BUILD) ]] || mkdir -p $(BUILD)

start-script: dir
	@echo '# build-time: $(shell date '+%Y-%m-%d-%H%M%S')' > $(START_PATH)
	@echo 'set -e' >> $(START_PATH)
	cd $(CERT_D) && ./scripts/key-admin --admin --$(ENV) --no --show-vars --append -o $(PWD)/$(START_PATH)
	@echo './$(APP) "$$@"' >> $(START_PATH)
	@chmod 755 $(START_PATH)

build: dir start-script
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BIN) .
.PHONY: build

bundle: build
	tar zcvf $(TARBALL) -C $(BUILD) .
deploy: bundle
	dp scp $(ENV) $(HOST_NUM) -- $(TARBALL) .

run: deploy
	echo "Please remember to run 'make clean' after this"
	dp ssh $(ENV) $(HOST_NUM) -- bash -c "echo On $(HOST_NUM) && tar zxf $(TARBALL) && ./$(START_SH)"

clean: clean-local clean-remote

clean-local:
	rm -r $(BUILD) $(TARBALL)

clean-remote:
	dp ssh $(ENV) $(HOST_NUM) -- rm -r $(TARBALL) $(START_SH) $(APP)/
