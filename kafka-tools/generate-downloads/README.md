# generate-downloads

`generate-downloads` is a utility to queue a message in Kafka which should trigger the full downloads to be rebuilt for a given published dataset, when they are missing.

## Configuration

You will need different values (and possibly more env vars) than shown in the examples and/or `Makefile`, below, see [config](./main.go) for the defaults.

## Queueing the message

Queue the message _inside the environment_ (or - deprecated - _locally_ using an ssh tunnel).

### Run in the environment

Run the following on your laptop, from this directory, to upload and run the program inside the environment (change the values to suit):

```shell
$ make INSTANCE_ID="xb1ae3d1-913e-43e0-b4c9-2c741744f12" DATASET_ID="weekly-deaths-local-authority" VERSION=2 EDITION=2021 ENV=prod SERVICE=cmd
# ...
```

The `INSTANCE_ID`, `DATASET_ID`, `EDITION` and `VERSION` used below should be provided by the publishing team who raised the problem.

- `DATASET_ID`, `EDITION` and `VERSION` can be taken from the URI of the dataset: `https://www.ons.gov.uk/datasets/{DATASET_ID}/editions/{EDITION}/versions/{VERSION}`
- If `INSTANCE_ID` is not provided, see section below [Getting the INSTANCE_ID, a worked example](#getting-the-instance_id-a-worked-example)
- `SERVICE` can either be 'cmd' or 'cantabular', depending what event you are re-triggering

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

You can now use Mongo or curl to obtain the ID.

### Using mongo

You can access mongodb using either

- your local client
  - see [accessing mongodb using robo3t](https://github.com/ONSdigital/dp-operations/blob/main/guides/mongodb.md)
  - select the `datasets` database
  - then [query mongodb](#query-mongodb), below
- or login to a host in the env and run the `mongo` client

#### Login to the env to use mongo client

First, get the mongodb host, user and password for the datasets database. You can find them, for example, by decrypting the [secrets for dp-dataset-api-web](https://github.com/ONSdigital/dp-configs/blob/main/secrets/prod/dp-dataset-api-web.json.asc).

Then, log into the box in production that has the `mongo` command available:

```shell
dp ssh prod publishing 10
```

and run the following command using the credentials decrypted from the secrets. Please note there is a space at the beginning of the line so that the password is not left in the history. :warning:

```shell
 mongo datasets --tls --host <MONGO_HOST> --username <MONGO_USER> --password <MONGO_PASSWORD> --tlsCAFile "/etc/docdb/rds-combined-ca-bundle.pem"
```

_NOTE: If boxes in Prod are rebuilt the number `10` used above may change and you will need to try other numbers and re-enter the mongo command until you get the mongo prompt._

You should see this prompt:

`rs0:PRIMARY>`

#### Query mongodb

You can enter, for example, the following command. Adapt it to your needs by adjusting the `links.dataset.id`, `links.edition.id` and `links.version.id` to the relevant values provided in the support ticket:

```javascript
db.getCollection('instances').find({"state":"published", "links.dataset.id":/weekly-deaths-local-authority/, "links.edition.id":"2022", "links.version.id":"32"})
```

It will return the required instance and you can now take a note of its `"id"`.

### Using the dataset API

Login to the `publishing_mount` host: `dp ssh prod publishing_mount 1`
and (on that host) you will need to be root (type `sudo -s`).

Find, and source (into that shell) the secrets for zebedee:

```shell
cd /var/lib/nomad/alloc/*/zebedee/local
. ./vars
```

Now, use `curl` to query the dataset API to obtain the required ID
(the below command uses the env var `$SERVICE_AUTH_TOKEN` which has been set above, so that part is copied _verbatim_, change only the URL)

```shell
curl -s -H "Authorization: Bearer $SERVICE_AUTH_TOKEN" localhost:10400/datasets/weekly-deaths-local-authority/editions/2022/versions/32 | jq .
```

confirm that that the above shows the correct document, and note its `"id"`, or extract that ID with the below:

```shell
curl -s -H "Authorization: Bearer $SERVICE_AUTH_TOKEN" localhost:10400/datasets/weekly-deaths-local-authority/editions/2022/versions/32 | jq -r .id
```
