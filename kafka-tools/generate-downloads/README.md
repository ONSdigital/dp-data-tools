# generate-downloads

`generate-downloads` is a utility to queue a message in Kafka which should trigger the full downloads to be rebuilt for a given published dataset, when they are missing.

## Configuration

You will need different values (and possibly more env vars) than shown in the examples and/or `Makefile`, below, see [config](./main.go) for the defaults.

## Queueing the message

Queue the message _inside the environment_ (or - deprecated - _locally_ using an ssh tunnel).

### Run in the environment

To run the program inside the environment (change the values to suit):

```shell
$ make INSTANCE_ID="xb1ae3d1-913e-43e0-b4c9-2c741744f12" DATASET_ID="weekly-deaths-local-authoritay" VERSION=2 EDITION=2021 ENV=prod
# ...
```

The above `make` does the following:

- creates a script with the required secrets (auth to MSK) (first asset)
- builds the binary (second asset)
- `dp scp` the assets onto the env
- `dp ssh` onto the env to run the assets
- cleans up (the env) on success
  - if it fails, you should tidy up with: `make clean-deploy ENV=...`

### Run locally

:warning: DEPRECATED. :warning:

Alternatively, run locally:

- ssh tunnel to kafka (if targeting a cloud environment)

```shell
$ make run # same make vars as running in env above
...
$ make clean # clean up local files
```
