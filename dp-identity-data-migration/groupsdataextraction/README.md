# dp-identity-api Zebedee Group Migration

## Description

When we put the new auth service live we will need to migrate all groups from the existing login mechanism (zebedee) to the new identity API. In order to do this we will need a scripted and reliable approach to exporting the groups and the list of members of those groups from zebedee.

It is important the userdataextraction and groupsdataextraction are ran in order:

1. [userdataextraction](../userdataextraction/README.md)
2. groupsdataextraction

## Export Script

![dataflow](dataflow.drawio.svg)

### Requirements

1. dp-cli access to required environment
2. florence/zebedee user and password for the required environment

### Set Up and Execution

Two terminal windows are required  one for the tunnel, another to run extracts.

1. Set Up and Run Tunnel
    If using localhost start the apps required to run local florence/zebedee (There is no need to start tunnel).
    If using an remote environment version

    ```shell
    dp remote allow <environment>
    dp ssh <environment> publishing 1 -p 10050:10050
    ```

2. In the other Terminal Widow set the Environment Variables :-

    ```shell
    export environment=<'localhost' 'develop' 'prod' 'production' 'sandbox'>
    if environment = localhost 
        export zebedee_host="http://localhost:8082"
        export email_domains="gmail.com,ons.gov.uk,ext.ons.gov.uk,methods.co.uk"
    else 
        export zebedee_host="http://localhost:10050"
        export email_domains="ons.gov.uk,ext.ons.gov.uk"

    export zebedee_user=<zebedee user admin email>
    export zebedee_pword=<zebedee user admin password for environment>
    export groups_filename="groups_export_$(date '+%Y-%m-%d_%H_%M_%S').csv"
    export groupusers_filename="groupusers_export_$(date '+%Y-%m-%d_%H_%M_%S').csv"
    export validusers_filename=<valid_users_export_$(date '+%Y-%m-%d_%H_%M_%S').csv>
    export invalidusers_filename=<invalid_users_export_$(date '+%Y-%m-%d_%H_%M_%S').csv>
    export s3_bucket=<s3_bucket>
    export s3_region="eu-west-2"
    export aws_profile=<profile name for environment>
    ```

3. Run the code

   ```shell
   go run dp-identity-data-migration/groupsdataextraction/group_extraction.go
   ```

### Output

The terminal logs and files are uploaded to S3 - by default the log file is also removed from your local system. To retain the log file for debugging:

```sh
  export debug=true
```

#### Terminal

```shell
This is for <environment>
=========  <groups_filename> file validiation =============
Expected row count: -  24
Actual row count: -  24
=========
Uploading <groups_filename> to s3
file uploaded to, https://<s3_bucket>.s3.<s3_region>.amazonaws.com/<environment>/<groups_filename>
Uploaded <groups_filename> to s3
---
list of 'users' that fail validation ....
---
=========  <groupusers_filename> file validation =============
Expected row count: -  168
Actual row count: -  168
=========
Uploading <groupusers_filename> to s3
file uploaded to, https://<s3_bucket>.s3.<s3_region>.amazonaws.com/<environment>/<groupusers_filename>
Uploaded <groupusers_filename> to s3
```

#### Files

This script creates 2 csv files.

#### groups csv format

group_name | user_pool_id | description | role_arn | precedence | last_modified_date | creation_date
--- | --- | --- | --- | --- | --- | ---
zebedee group ID | empty | zebedee group name | empty | default value 10 | empty | empty

#### usergroup csv format

ser_name | group_name
--- | ---
user email | group or role names (comma separated list as string)

### Import script

```sh
cd ../dp-identity-api/scripts/import_users/groups/

export FILENAME="<ENVIRONMENT>/<filename>"
export S3_BUCKET=<s3_bucket>
export S3_BASE_DIR=""
export S3_REGION=<s3_region>
export USER_POOL_ID=<cognito_user_pool_id>

go run restore_groups.go
```

**Note** *don't forget to unset the environmental variables that had been set during the export*
