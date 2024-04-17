## Find Content Items

This script is designed to search for specific types of content items in the directories and subdirectories of zebedee’s filestore. Specifically, the script is looking for `data.json` or `data_cy.json` files, counting the content types (i.e. `bulletin`, `timeseries`, `article` etc.) and then displaying how many of each type were found, along with the time taken for the script to run. 


### Filtering by Type: 
The script provides options to filter the search results by type, and the script will only display the counts of the filtered value once finished.

### Latest Release Only: 
Optionally, you can choose to include only the latest release of each content item. As with the `type` filtering, the script will only display counts of the latest release of all content types. This can be used in conjunction with the filtering by `type`.

### Display Statistics: 
After the search is complete, the script will display the count for each the found content items. This gives you a quick overview of the distribution of content items based on their types.

## Prerequisites

- a functioning dp tool from dp-cli
    - dp scp will be used to put the consumer binary into the env
    - then dp ssh to run it in the env

To build the tool in a remote environment, you will need to run the following `make` command, where env is either `sandbox`, `staging` or `prod` :

    ```shell
    make deploy <env>
    ```

The above command will create the binary in the root of `publishing_mount 1`, `publishing_mount 2`, `web_mount 1` and `web_mount 2` in the selected env.

You will then need to ssh into the desired env (`sandbox` etc.), subnet (`web` or `publishing`) and box (`1` or `2`)

For example -

```
dp ssh sandbox publishing 1 
```



## How to run the script

Depending on which subnet you are in (i.e. `web` or `publishing`) you will need to run the tool in the specific directory where the content data is stored.

For `publishing` - 

```
cd /var/florence/zebedee/master
```

For `web` - 

```
cd /var/babbage/site
```

### Command-line Flags

As specified above, you have the option to include flags which will filter the data returned by the script.

- `filter`: Specify the type to filter the search results. If you're interested in a specific type of content item, you can use this flag to narrow down the results.
- `latestrelease`: Include only the latest release of each content item. Use this flag when you're interested only in the most recent data.

To run the script without any flags, use the following command -

```

```
To run the script with the flags, use the following command. Either of the flags can be removed as required - 

```
-filter=bulletin -latestrelease=true
```

