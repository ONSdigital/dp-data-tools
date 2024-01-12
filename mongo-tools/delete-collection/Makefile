MONGO_SHELL = mongo
DB_NAME ?= does-not-exist
COLLECTION_NAME ?= does-not-exist
MONGODB_USERNAME ?= root

.PHONY: delete-collection

delete-collection:
	@read -sp "If you are sure you want to delete collection $(DB_NAME)/$(COLLECTION_NAME), please enter the correct $(MONGODB_USERNAME) password: " password; \
	if [ "$$password" != "" ]; then \
		$(MONGO_SHELL) --tlsCAFile ../../../dp-setup/ansible/roles/docdb/files/rds-combined-ca-bundle.pem --tlsAllowInvalidHostnames --tls \
		  -u $(MONGODB_USERNAME) -p "$$password" --eval "const DB_NAME = '$(DB_NAME)'; const COLLECTION_NAME = '$(COLLECTION_NAME)'" delete-collection.js  ; \
	else \
		echo "Deletion aborted"; \
	fi