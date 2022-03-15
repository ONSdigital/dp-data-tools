# Kafka topic drain

Using TLS kafka requires us to authenticate to kafka to perform any action.
This script attempts to make this as easy as possible, by using an app's
certificate when running one of the dp-kafka utils.

To drain a topic in MSK, we use a
[dp-kafka example consumer](https://github.com/ONSdigital/dp-kafka) as our app.

The defaults for the process below will drain the `observation-imported` topic
(for the `dp-observation-importer` app) on the `develop` env. Follow these steps:

## Prerequisites

You will need:

* `dp-configs` for the app's cert (used to authenticate to kafka as the app).
* `dp-kafka` for the example consumer. We will build this locally.
* a functioning `dp` tool [from dp-cli](https://github.com/ONSdigital/dp-cli)
  * `dp scp` will be used to put the consumer binary into the env
  * then `dp ssh` to run it in the env

Get them up-to-date:

```bash
cd ~/src/github.com/ONSdigital/dp-configs # wherever you keep this
git switch master && git pull             #   get up-to-date

cd ~/src/github.com/ONSdigital/dp-kafka   # wherever you keep this
git switch main   && git pull             #   get up-to-date

cd ~/src/github.com/ONSdigital/dp-cli     # wherever you keep this
git pull && make install                  #   get up-to-date
```

Return to this directory to continue.

## Steps

### Turn off other topic consumers

First, stop the app(s) in nomad to turn off any consumers of the topic
for this consumer group.

This is because we need our process to consume from (drain) all partitions
for this consumer group.

In this example (i.e. the default consumer of the topic),
we will stop the `dp-observation-importer` app:

In nomad choose Jobs --> `dp-observation-importer` and click `Stop`.

### Drain the `observation-imported` topic

The makefile `kafka-tools/drain-topic/Makefile` tries to keep this as simple as possible:

```bash
make drain ENV=develop       # or ENV=production
```

Hit **Ctrl-C** to stop the consumer once it acquiesces.
(i.e. no more messages to consume)

## Clean up

This will remove remote files (in the env) and local build files.

```bash
make clean ENV=develop       # tidy up after yourself, please!
```

### In nomad, restart the stopped service

Restart the app to return normality. Check all's well.

---

## Example targetting `import-observations-inserted` topic

*Alternatively*, you might wish to drain a *non-default* topic,
say, `import-observations-inserted`.

We will need to use the correct consumer group for the topic
(and use the cert/secrets from the corresponding consumer app).

For this example topic, the consumer app is: `dp-import-tracker`,
and the *drain* step (as part of the above [steps](#steps)) would therefore be:

```bash
make drain TOPIC=import-observations-inserted GROUP=dp-import-tracker ENV=develop       # or ENV=production
```

Hit **Ctrl-C** to stop the consumer once it acquiesces.

Remember to [clean up](#clean_up)