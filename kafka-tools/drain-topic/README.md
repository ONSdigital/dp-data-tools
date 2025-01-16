# Kafka topic drain

Using TLS kafka requires us to authenticate to kafka to perform any action.
This script attempts to make this as easy as possible, by using an app's
certificate when running one of the dp-kafka utils.

To drain a topic in MSK, we use a
[dp-kafka example consumer](https://github.com/ONSdigital/dp-kafka) as our app.

The defaults for the process below will drain the `observation-imported` topic
(for the `dp-observation-importer` app) on the `sandbox` env. Follow these steps:

## Prerequisites

You will need:

* `dp-configs` for the app's cert (used to authenticate to kafka as the app).
* `dp-kafka` for the example consumer. We will build this locally.
* a functioning `dp` tool [from dp-cli](https://github.com/ONSdigital/dp-cli)
  * `dp scp` will be used to put the consumer binary into the env
  * then `dp ssh` to run it in the env

Get them up-to-date (for example):
```bash
cd ~/src/github.com/ONSdigital/dp-configs # wherever you keep this
git switch main && git pull               # get up-to-date

cd ~/src/github.com/ONSdigital/dp-kafka   # wherever you keep this
git switch main && git pull               # get up-to-date

cd ~/src/github.com/ONSdigital/dp-cli     # wherever you keep this
git pull && make install                  # get up-to-date
```

Return to this directory to continue.

## Steps

### Turn off other topic consumers

First, stop the app(s) in nomad to turn off any consumers of the topic
for this consumer group.

This is because we need our process to consume from (drain) all partitions
for this consumer group:

In nomad choose Jobs --> `dp-observation-importer` (or the relevant consumer app name) and click `Stop`.

### Drain the topic 

The makefile `kafka-tools/drain-topic/Makefile` tries to keep this as simple as possible.

To drain the *default* topic (the `observation-imported` topic):

```bash
make drain ENV=sandbox       # or ENV=prod
```

*Alternatively*, we might wish to drain a *non-default* topic,
say, `import-observations-inserted`.

We will need to use the correct consumer group for the topic
(and use the cert/secrets from the corresponding consumer app).

For the `import-observations-inserted` topic, the consumer app is: `dp-import-tracker`,
and the *drain* step would therefore be:

```bash
make drain TOPIC=import-observations-inserted GROUP=dp-import-tracker ENV=sandbox       # or ENV=prod
```

Hit **Ctrl-C** to stop the consumer once it acquiesces (i.e. no more messages to consume).

### Clean up

This will remove files that were copied onto the environment by running the `make drain` target and any files built locally from the same command:

```bash
make clean ENV=sandbox       # or ENV=prod
```

### In nomad, restart the stopped service

Restart the app to return normality. Check all's well.
