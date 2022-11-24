# ami-check

This little utility reports the list of ami's we have by environment and sorted by time of creation.
If also produces a count and list of all the unique AMI's for possible future processing.
You need to have Prod access for this to work.

## Run it in its directory with

```shell
./get-ami-info.sh
```

NOTE: The above will pop open in your browser confirmation for each AWS environment.
