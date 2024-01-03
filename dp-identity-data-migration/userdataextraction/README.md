# dp-identity-api Zebedee User Migration

## Description

When we put the new auth service live we will need to migrate all users from the existing login mechanism (zebedee) to the new identity API. In order to do this we will need a scripted and reliable approach to exporting the users from zebedee.

It is important the userdataextraction and groupsdataextraction are ran in order:

1. userdataextraction
2. [groupsdataextraction](../groupsdataextraction/README.md)

## Extraction Script

![dataflow](dataflow.drawio.svg)

### Requirements

1. dp-cli access to required environment
2. florence/zebedee user and password for the required environment

### Set Up and Execution

Two terminal windows are required - one for the tunnel, another to run extracts.

1. Set Up and Run Tunnel
    If using localhost start the apps required to run local florence/zebedee (There is no need to start tunnel).
    If using a remote environment version:

    ```shell
    dp remote allow <environment>
    dp ssh <environment> publishing 1 -p 10050:10050
    ```

2. In the other Terminal Window, set the required Environment Variables :-

    ```shell
    export environment=<'localhost' 'sandbox' 'staging' or 'prod'>
    # if environment == localhost
        export zebedee_host="http://localhost:8082" 
        export email_domains="gmail.com,ons.gov.uk,ext.ons.gov.uk,methods.co.uk"
    # else
        export zebedee_host="http://localhost:10050" 
        export email_domains="ons.gov.uk,ext.ons.gov.uk"

    export zebedee_user=<zebedee user admin email>
    export zebedee_pword=<zebedee user admin password for environment>
    extract_date=$(date '+%Y-%m-%d_%H_%M_%S')
    export groups_filename="groups_export_$extract_date.csv"
    export groupusers_filename="groupusers_export_$extract_date.csv"
    export validusers_filename="valid_users_export_$extract_date.csv"
    export invalidusers_filename="invalid_users_export_$extract_date.csv"
    export s3_bucket=<s3_bucket>
    export s3_region="eu-west-2"
    export aws_profile=<profile name for environment>
    ```

3. Run the code

   ``` shell
   go run user_extraction.go
   ```

### Output

The terminal logs and files are uploaded to S3 - by default the log file is also removed from your local system. To retain the log file for debugging:

```sh
  export debug=true
```

#### Terminal

Example output from localhost:

```shell
========= file validiation =============
Expected row count: -  2
Valid users row count: -  1
Invalid users row count: -  1
=========
========= Uploading valid users file to S3 =============
file uploaded to, https://<s3_bucket>.s3.<s3_region>.amazonaws.com/<environment>/<filename>
file uploaded to, https://<s3_bucket>.s3.<s3_region>.amazonaws.com/<environment>/invalid_<filename>
========= Uploaded files to S3 =============
```

#### Files

This script creates 2 csv files one with users with valid emailIds and other with invalid users.

#### csv file format

cognito:username | name | given_name | family_name | middle_name | nickname | preferred_username | profile picture | website | email | email_verified | gender | birthdate | zoneinfo | locale | phone_number | phone_number_verified | address | updated_at | cognito:mfa_enabled
--- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | ---
uudi | --- | from email if expected format | from email if expected format | --- | --- | --- | --- | --- | email | true | --- | --- | --- | --- | --- | false | --- | --- | false

### Import script

To run the import:

```shell
cd ../dp-identity-api/scripts/import_users/users/

export FILENAME="<ENVIRONMENT>/<filename>"
export S3_BUCKET=<s3_bucket>
export S3_BASE_DIR=""
export S3_REGION=<s3_region>
export USER_POOL_ID=<cognito_user_pool_id>

go run restore_users.go
```

**Note** *don't forget to unset the environmental variables that had been set during the export stage*
