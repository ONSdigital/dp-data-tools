# Load testing Cantabular Publishing servers in Staging to exercise its ASG

The files in this folder `autocannon-go-2` have been derived from:

https://github.com/GlenTiki/autocannon-go

NOTE: the original files build and run, but doing `go mod tidy` throws up an issue ... which persists in the derived code.

## Purpose

The derived code is tunned to make sufficient requests to the Cantabular load balancer endpoint for enough time to see close to 100% CPU usage that will cause the ASG to add more servers to its ASG.

The new code cyles through a list of codebokk endpoints to get the Cantabular servers to load all of the data from the databases into memory to achieve maximum utilisation / performance from the Cantabular servers as advised by Sensible code.

## Pre-requsites

The Cantabular servers will need to have the ansible code loading up the following list of files (in file ansible/inventories/staging/group_vars/all):

```yaml
cantabular_publishing_data_files:
  - d_cantabf_v10-1-0_recipe_usual-res_v1-0_drl_ts-dm_usual-res_oa_20220810.zip
  - ons_phase2-household_reference_persons.dat
  - ons_phase2-person_household.dat
  - ons_phase2-person_usual_residents.dat
  - t_cantaba-v10-0-0_ar2776-c21ew_metadata-version6-0_cantab_220622-1_household_p1-ts1-i1_1_20220707.dat
  - t_cantabf_v10-0-0_recipe_hh_v1-1_drl_ts-housing_hh_LA_20220616.dat
cantabular_publishing_metadata_files:
  - synth_meta.zip
```

## Getting the codebook endpoints to use in test code

With the above databases loaded:

log on to Staging publishing 3:

dp ssh staging publishing 3

add command `jq`

run commands to get info on what databases the cantabular server in publishing has loaded:

curl http://cantabular-server-publishing.internal.staging:14000/v10/datasets | jq .

curl http://cantabular-server-publishing.internal.staging:14000/v10/datasets | jq . | egrep name | grep -vwE "COUNT"

from the first one above, in what it outputs we pick the large and 'real' databases and get a list of each's codebook with:

curl 'http://cantabular-server-publishing.internal.staging:14000/v10/codebook/People-Households?cats=false' | jq .

curl 'http://cantabular-server-publishing.internal.staging:14000/v10/codebook/Usual-Residents?cats=false' | jq .

curl 'http://cantabular-server-publishing.internal.staging:14000/v10/codebook/Household-Ref-Persons?cats=false' | jq .

The output from the above is then built into the array `uriList` in the function `runClients()` in the structure you see in the code.

## Details of load test

How much load is applied is tunable, but requires real observations to see effect of changing parameters within the Makefile and the file autocannon.go

Load is applied to the Cantabular servers in Staging Publishing by having an instance of the autocannon.go built executable (for AWS machines) run from one or more of the publishing boxes in Staging (so as to be able to access the Cantabular Servers in Staging Publishing network).

The load test is observed by configuring the Cantabular Publishing Server ASG in staging to have a min of three machines and a max of six (instance type r5.large, at time of initial testing), and then logging into each of the 3 servers in seperate terminals and running `htop` in each.

## Running load test

The load test is then run by simply doing: `make`

This will build the autocannon.go app to run on linux, load it into  1 or more publishing boxes (as determined by `host_numbers` list in the Makefile) and cause each of these apps to start running some minutes later as determined by lines like this in the Makefile:

```makefile
TZ=UTC; start_time=$(shell TZ=UTC date -v +3M '+%H:%M'); \
```

(where the `+3M` is three minutes, which is OK for 4 instances of autocannon.go being deployed ... This `+3M` needs changing to `+4M` for six instances and needs to be `+5M` for 8 instances)

The time delay for execution gives the makefile enough time to sequentially issue the commands to each publishing box, and then the apps all run at the same time to apply load.

## Adjusting amount of load

The load applied can be increased or decreased in a number of ways, where the aim is to first run the load test with the `make` command and observe the current load via `htop` for 3 servers over the duration of the test.

Next we adjust the value for `pipelining` in this line of code:

```go
pipeliningFactor := flag.Int("pipelining", 6, "The number of pipelined requests to use.")
```

and re-run tests to observe the effect on CPU load.

You can also affect the load by adjusting the number of instances of the load test running by altering the list `host_numbers` in the makefile together with the delay needed in lines like this in the makefile:

```makefile
TZ=UTC; start_time=$(shell TZ=UTC date -v +3M '+%H:%M'); \
```

as detailed in section `Running load test` above.

The initial aim of adjustments is to see 100% load on three servers for the duration of the test.

The duration of test run can be altered in this line:

```go
runtime := flag.Int("duration", 75, "The number of seconds to run the autocannnon.")
```

Every time you make any adjustment, re-run the load test and observe (and write down) your observations together with the values of any parameters you have changed because its easy to run many different tests and loose track of what changes affect the load observations.

Tuning load test is not a straightforward thing to do and requires experimentation.

It becomes tricky as you apply sufficient load for the ASG to scale in 1 or more extra servers.

## Summary

Remember the purpose of load testing is to demonstrate that the Auto scaling pulls in extra servers after the load has been sufficiently high for a while and keeps them there for a set level of load and then after some time when the load is reduced, then number of servers is scalled back.
