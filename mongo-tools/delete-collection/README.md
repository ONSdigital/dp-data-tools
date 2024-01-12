# DocumentDB Collection Deletion Script

## Overview

This script is designed to facilitate the deletion of a specified collection from a DocumentDB database within our infrastructure. It involves connecting to MongoDB via SSH, decrypting the root password, and executing the deletion process through a Makefile command.

## Prerequisites

Before using this script, ensure that the following prerequisites are met:

1. Make sure that you're signed in to the correct aws environment:

```shell
aws sso login --profile=dp-<name-of-environment>
```

2. **SSH Access to MongoDB**

3. **Vault Access**

## Steps

***
:warning: Deleting a collection is irreversible. Ensure that you have a backup or are absolutely certain before proceeding. :warning:
***


### 1. SSH into MongoDB

Use the provided `dp` command to establish an SSH connection to the MongoDB server:

```shell
dp ssh <name-of-environment> publishing 4 -p 27017:$MONGO_BIND_ADDR:27017
```

You can find the `MONGO_BIND_ADDR` from the `dp-configs` secrets file of any app that usses this environment varibale. Make sure you choose the correct environment, i.e. `sandbox`, `staging`, or `prod`.


### 2. Decrypt Root Password

Decrypt the root password via `dp-setup` using the following command:

   ```shell
   cd dp-setup/anisble
   export ONS_DP_ENV=<name-of-environment>
   yq .docdb_login_password inventories/$ONS_DP_ENV/group_vars/all | ansible-vault decrypt --vault-id=$ONS_DP_ENV@.$ONS_DP_ENV.pass
   ```


### 3. Delete Collection

Navigate to the `dp-data-tools/mongo-tools/delete-collection` directory:

```
cd dp-data-tools/mongo-tools/delete-collection
```


Execute the following Makefile command:


```
make delete-collection DB_NAME=<database_name> COLLECTION_NAME=<collection_name>
```

This command will prompt you to enter the root password that you decrypted via the vault command above. Once provided, the specified collection will be deleted.

Note: Replace `<database_name>` and `<collection_name>` with the actual DocumentDB database nameand collection name, respectively.

### Makefile Variables:

`DB_NAME`: The name of the MongoDB database from which the collection will be deleted.

`COLLECTION_NAME`: The name of the collection to be deleted.

`MONGODB_USERNAME`: The MongoDB username (default: root).


### .js File:

The actual deletion of the collection is performed by the `delete-collection.js` script. It drops the specified collection and provides feedback on the success or failure of the operation.