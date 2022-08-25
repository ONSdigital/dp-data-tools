# generate-downloads

`generate-downloads` is a utility to queue a message in Kafka which should trigger the full downloads to be rebuilt for a given published dataset, when they are missing.

## Configuration

You will need different values (and possibly more env vars) than shown in the examples and/or `Makefile`, below, see [config](./main.go) for the defaults.

## Queueing the message

Queue the message _inside the environment_ (or - deprecated - _locally_ using an ssh tunnel).

### Run in the environment

To run the program inside the environment (change the values to suit):

```shell
$ make INSTANCE_ID="xb1ae3d1-913e-43e0-b4c9-2c741744f12" DATASET_ID="weekly-deaths-local-authority" VERSION=2 EDITION=2021 ENV=prod
# ...
```

The `INSTANCE_ID`, `DATASET_ID`, `EDITION` and `VERSION` used below should be provided by the publishing team who raised the problem.
- `DATASET_ID`, `EDITION` and `VERSION` can be taken from the URI of the dataset: `https://www.ons.gov.uk/datasets/{DATASET_ID}/editions/{EDITION}/versions/{VERSION}`
- If `INSTANCE_ID` is not provided, see section below [Getting the INSTANCE_ID, a worked example](#getting-the-instance_id-a-worked-example)

The above `make` does the following:

- creates a script with the required secrets (auth to MSK) (first asset)
- builds the binary (second asset)
- `dp scp` the assets onto the env
- `dp ssh` onto the env to run the assets
- cleans up (the env) on success
  - if it fails, you should tidy up with: `make clean ENV=...` :warning:

### Run locally

:warning: DEPRECATED. :warning:

Alternatively, run locally:

- ssh tunnel to kafka (if targeting a cloud environment)

```shell
$ make run # same make vars as running in env above
...
$ make clean # clean up local files
```

## Getting the `INSTANCE_ID`, a worked example

In the support ticket that is missing the `INSTANCE_ID`, the publishing team had provided this URI:

`https://www.ons.gov.uk/datasets/weekly-deaths-local-authority/editions/2022/versions/32`

First, get the mongodb host, user and password for the datasets database. You can find them, for example, by decrypting the [secrets for dp-dataset-api-web](https://github.com/ONSdigital/dp-configs/blob/main/secrets/prod/dp-dataset-api-web.json.asc).

Then, log into the box in production that has the `mongo` command available:

```shell
dp ssh prod publishing 10
```

and run the following command using the credentials decrypted from the secrets. Please note there is a space at the beginning of the line so that the password is not left in the history.

```shell
 mongo datasets --tls --host <MONGO_HOST> --username <MONGO_USER> --password <MONGO_PASSWORD> --tlsCAFile "/etc/docdb/rds-combined-ca-bundle.pem"
```

_NOTE: If boxes in Prod are rebuilt the number `10` used above may change and you will need to try other numbers and re-enter the mongo command until you get the mongo prompt._

Once you get this prompt:

`rs0:PRIMARY>`

You can enter for example the following command. Adapt it to your needs by adjusting the `links.dataset.id`, `links.edition.id` and `links.version.id` to the relevant values provided in the support ticket:

```shell
db.getCollection('instances').find({"state":"published", "links.dataset.id":/weekly-deaths-local-authority/, "links.edition.id":"2022", "links.version.id":"32"})
```

It will return the required instance and you can now take a note of its `"id"`.
