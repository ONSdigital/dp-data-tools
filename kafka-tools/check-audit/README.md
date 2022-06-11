# Check-audit

This utility:

* consumes messages from the `audit` kafka topic
* builds a list of accessed paths and their respective results (as `successful` and `unsuccessful`
counters) stored in-memory

The results (a list of actions) are displayed/logged when you hit the `<Enter>` key (to terminate the app).

## How to run the utility locally

Run:

* `go build`
* `HUMAN_LOG=1 KAFKA_ADDR=$BROKERS KAFKA_VERSION=2.6.1 ./check-audit

Where `$BROKERS` should be a comma-separated list of broker addresses -
each value in the format `localhost:9092` (i.e. `<ip-address>:<port>`).

Adjust the value of `KAFKA_VERSION` to suit your local environment.

## How to run the utility on an environment

In this directory, run

```shell
$ make run ENV=$ENV_NAME
#   e.g.
$ make run ENV=sandbox
## Hit <Enter> to terminate the app
```

this will do the following:

* build the app (from the source)
* obtain the current settings (kafka address, version and client cert/key using `key-admin`)
* put the above settings into a shell script
* copy (scp) the app and script to a box in the env
* ssh onto that box (in the env) and run the script (which sets the vars and runs the binary)

Hit `<Enter>` to terminate the app. :warning:

### Clean up

To clean-up files (including those on the target host), run

```shell
$ make clean ENV=$ENV_NAME
...
```

## Note about HUMAN_LOG and special characters encoding

If you run check-audit without HUMAN_LOG, then any special character will be presented as a unicode escaped ASCII hexadecimal code.
For example, if an audit event contains `query_param` with value `k1=v1&k2=v1`, the value will be displayed as `k1=v1\u0026k2=v2` unless the tool is run with HUMAN_LOG=1
