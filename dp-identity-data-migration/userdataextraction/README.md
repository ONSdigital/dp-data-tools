## zebedee user migration

This script creates a csv file that is to be loaded into aws cognito 
it is designed to run on local machine  using dp-cli.

### How to run the utility

1) set up local environment variables where you are going to run the migration 
```
export filename="/Users/ann/dp-identity-api-zebedee-user-extraction.csv"
```

for local host 
export zebedee_host="http://localhost:8082"
for all other environments
export zebedee_host="http://localhost:10050"


export zebedee_pword=<your florence password for the environment>  
export zebedee_user=<your user name for florence>

2) create the tunnel 
if not running for local run the following dp-cli commands 
dp remote allow <environment>

then create a tunnel to the required 'box'

dp ssh develop publishing 1 -p 10050:10050

3) run prog from dp-data-tools directory...
go run dp-identity-data-migration/user_extraction.go

it will list the users from zebedee 
the process will stop if there is an issue i.e. not set environment variables, a problem creating the output file.

4) once run unset zebedee_pword