#zebedee user migration
## Groups Migration

### Description

When we put the new auth service live we will need to migrate all groups from the existing login mechanism (zebedee) to the new identity API. In order to do this we will need a scripted and reliable approach to exporting the groups and the list of members of those groups from zebedee.

###What
Investigate best approach to export all groups and their list of users from zebedee ready for import into Cognito
There is a GET /teams endpoint on zebedee that lists all the teams and their members
We could also consider loading from the /var/florence/zebedee/teams directory where the JSON files have the same info
The script needs to output to things:
1. A list of all groups
2. A list of users and for each user a list of groups they are in
    for each user extract their current permissions
    (Due to complex structure proposed a single record for csv output is being created for each user and group/permission)

![dataflow](dataflow.drawio.svg)

### Solution 
####Requirements 
1.  see go.mod 
2.  dp-cli access to required environment
3.  The programs use dp-zebedee-sdk-go to access and extract from zebedee

4.  Environmental Variables :-
    <code>export zebedee_user=\<zebedee user email\></code>
    <code>export zebedee_pword=\<zebedee user password for environment\></code>
    <code>export zebedee_host=\<local "http://localhost:8082"; otherwise "http://localhost:10050"\></code>
    <code>export groups_filename=\<full path to file\></code>
    <code>export groupusers_filename=\<full path to file\></code>

5.  If using localhost start the apps required to run florence/zebedee 
    There is no need to start tunnel
    If using an remote environment version
    in a separate terminal run:-
        <code> dp remote allow \<environment>\</code>
        <code> dp ssh develop publishing 1 -p 10050:10050</code>

6. Run the code....
    <code>go run dp-identity-data-migration/user_extraction.go</code>


This script creates csv files that are to be loaded into aws cognito it is designed to run on local machine using dp-cli.
output 
group csv 
group_name | user_pool_id | description | role_arn | precedence | last_modified_date | creation_date
--- | --- | --- | --- | --- | --- | ---
zebedee group name | empty | zebedee group name | empty | default value 10 | empty | empty 


How to run the utility
set up local environment variables where you are going to run the migration
export filename=<group file name> (it is recomended to use full path )



create the tunnel if not running for local run the following dp-cli commands dp remote allow
then create a tunnel to the required 'box'

dp ssh develop publishing 1 -p 10050:10050

run prog from dp-data-tools directory... go run dp-identity-data-migration/user_extraction.go
it will list the users from zebedee the process will stop if there is an issue i.e. not set environment variables, a problem creating the output file.

once run unset zebedee_pword