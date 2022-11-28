# ami-check

This little utility reports the list of ami's we have by environment and sorted by time of creation and their usage status.
You need to have Prod access for this to work.

## Run it in its directory with

```shell
./ami-check.sh
```

This runs the three step scripts.

NOTE: The above will pop open in your browser confirmation for each AWS environment.

The final result file for dp-setup is called 'ami-used-status.txt' in the 'results' directory.
