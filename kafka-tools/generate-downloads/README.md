# generate-downloads

`generate-downloads` is a utility to queue a message in Kafka which should trigger the full downloads to be rebuilt for a given published dataset, when they are missing.

## Configuration

You will need different values (and possibly more env vars) than shown in the examples and/or `Makefile`, below, see [config](./main.go) for the defaults.

The `INSTANCE_ID` used below should be provided by the publishing team who raised to problem. If no one in publishing is available with the technical knowledge of how to get the `INSTANCE_ID`, see section below `Getting the INSTANCE_ID, a worked example`

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

In the support ticket that is missing the `INSTANCE_ID`,the publishing team had provided this URI:

`https://www.ons.gov.uk/datasets/weekly-deaths-local-authority/editions/2022/versions/32`

First decrypt the password for `docdb_login_password` in `dp-setup/ansible/inventories/prod/group_vars` for use below.

Then log into the box in production that has the `mongo` command available:

```shell
dp ssh prod publishing 10
```

and run this command (substituting the decrypted password where indicated):

```shell
mongo admin --tls --host prod-docdb-cluster.cluster-cqs87nhaxvnx.eu-west-2.docdb.amazonaws.com --username root --password <Place decrypted docdb_login_password here> --tlsCAFile "/etc/docdb/rds-combined-ca-bundle.pem"
```

NOTE: If the above command fails -

 1. Check the cluster address has not changed in AWS console

 2. If boxes in Prod are rebuilt the numer `10` used above may change and you will need to try other numbers and re-enter the `mongo ...` command until you get the mongo prompt

Once you get this prompt:

`rs0:PRIMARY>`

You can enter for example these two commands (adapt to your needs by adjusting the string `weekly-deaths-local-authority` to the equivalent in the URI provided in the support ticket URI that you have, you will also need to adjust the date to something like 5 days before the support ticket date):

```shell
use datasets

db.getCollection('instances').find({"state":"published", "last_updated":{$gt:new ISODate("2022-08-20T00:00:00")}, "links.dataset.id":/weekly-deaths-local-authority/})
```

and when this example was run on 25th Aug 2022 it produced two results.

Copy and paste the results into your editor (in this case, vscode and save as `inst.json`, so that we can right click in the document and select `Format Document` to easily identify the fields mentioned below).

From the support ticket URI we copy this sub string: `/datasets/weekly-deaths-local-authority/editions/2022/versions/32` and search for it in the two results.

The first instance of it that we are interested in should be in links.version.href and a little way before that we find the `INSTANCE_ID` as the `"id"` with value of (in this example): `"e581b0c9-8e99-44db-ad71-47fad689827a"` as per:

```json
    "id": "e581b0c9-8e99-44db-ad71-47fad689827a",
    "last_updated": ISODate("2022-08-23T08:51:02.456Z"),
    "e_tag": "16a25b6b85710f7ebbeeaa065eb36ec896af529a",
    "links": {
        "dataset": {
            "href": "http://10.30.154.142:10400/datasets/weekly-deaths-local-authority",
            "id": "weekly-deaths-local-authority"
        },
        "dimensions": {},
        "edition": {
            "href": "http://10.30.44.112:10400/datasets/weekly-deaths-local-authority/editions/2022",
            "id": "2022"
        },
        "job": {
            "href": "http://10.30.154.142:10700/jobs/26627f87-2ee9-467e-b44a-79a0bf679eaf",
            "id": "26627f87-2ee9-467e-b44a-79a0bf679eaf"
        },
        "self": {
            "href": "http://10.30.155.83:10400/instances/e581b0c9-8e99-44db-ad71-47fad689827a"
        },
        "version": {
            "href": "http://10.30.155.83:10400/datasets/weekly-deaths-local-authority/editions/2022/versions/32",
            "id": "32"
        }
    }
```

You can follow this process to extract the `"id"` for the `INSTANCE_ID` to use for the issue you are working on in other sections of this document.
