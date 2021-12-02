#dp-identity-api Zebedee User Migration
## Description

When we put the new auth service live we will need to migrate all users from the existing login mechanism (zebedee) to the new identity API. In order to do this we will need a scripted and reliable approach to exporting the users from zebedee.

##What
Investigate export of all users and transformation to the format required here: Creating the User Import .csv File - Amazon Cognito
We should likely validate the users' emails are @ons.gov.uk or @ext.ons.gov.uk emails and write these out to a separate list for admin review (but this is more of an implementation detail than spike one)
We are going to need to split the names to first and last as best as we can (does not need to be perfect as admin can fix any issues later, but we should consider if the email field can help with this at all) (but again this is probably more of an implementation detail)

![dataflow](dataflow.drawio.svg)

## Solution 
###Requirements 
1.  dp-cli access to required environment
2.  florence/zebedee user and password for the required environment

###Set Up and Execution
two terminal windows are required  one for the tunnel, another to run extracts 
1. Set Up and Run Tunnel
    If using localhost start the apps required to run local florence/zebedee (There is no need to start tunnel).
    If using an remote environment version
    ```shell
    dp remote allow <environment>
    dp ssh develop publishing 1 -p 10050:10050
    ```
3. In the other Terminal Widow 
    Set the require  Environmental Variables :-
    ``` shell 
    export zebedee_user=<zebedee user email>
    export zebedee_pword=<zebedee user password for environment>
    export zebedee_host=\<local "http://localhost:8082"; otherwise "http://localhost:10050">
    export filename=<full path to file>
    export s3_base_dir=<S3 base directory>
    export s3_bucket=<S3 bucket name>
    export s3_region=<S3 bucket region>
    export email_domains=<comma separated valid email domain ex: ons.com>

4. Run the code....
   ``` shell
   go run dp-identity-data-migration/userdataextraction/user_extraction.go
   ```

### Output
#### in Terminal 
```
=========  ...users.csv file validiation =============
Expected row count: -  112
Actual row count: -  112
=========
```

####Files
This script creates 2 csv files one with users with valid emailIds and other with invalid users 
#### csv file format 
cognito:username | name | given_name | family_name | middle_name | nickname | preferred_username | profile	picture | website | email | email_verified | gender | birthdate | zoneinfo | locale | phone_number | phone_number_verified | address | updated_at | cognito:mfa_enabled
--- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | ---
uudi | --- | from email if expected format | from email if expected format | --- | --- | --- | --- | --- | email | true | --- | --- | --- | --- | --- | false | --- | --- | false 


**Note** *don't forget to unset the environmental variables that had been set*
