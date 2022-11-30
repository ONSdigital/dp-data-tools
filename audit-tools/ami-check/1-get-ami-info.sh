#!/usr/bin/env bash

# get staging ami info
export ONS_DP_ENV=staging FLY_TARGET=ci AWS_DEFAULT_PROFILE=dp-staging AWS_DEFAULT_REGION=eu-west-2 ONS_DP_AWS_REGION=eu-west-2 AWS_PROFILE=dp-staging
unset AWS_DEFAULT_PROFILE AWS_DEFAULT_REGION ANSIBLE_REMOTE_USER

aws sso login --profile dp-staging

aws ec2 describe-images --owner self --output json | jq . >tmp/staging-amis.json

# get sandbox ami info
export ONS_DP_ENV=sandbox FLY_TARGET=ci AWS_DEFAULT_PROFILE=dp-sandbox AWS_DEFAULT_REGION=eu-west-2 ONS_DP_AWS_REGION=eu-west-2 AWS_PROFILE=dp-sandbox
unset AWS_DEFAULT_PROFILE AWS_DEFAULT_REGION ANSIBLE_REMOTE_USER

aws sso login --profile dp-sandbox

aws ec2 describe-images --owner self --output json | jq . >tmp/sandbox-amis.json

# get prod ami info
export ONS_DP_ENV=prod FLY_TARGET=ci AWS_DEFAULT_PROFILE=dp-prod AWS_DEFAULT_REGION=eu-west-2 ONS_DP_AWS_REGION=eu-west-2 AWS_PROFILE=dp-prod
unset AWS_DEFAULT_PROFILE AWS_DEFAULT_REGION ANSIBLE_REMOTE_USER

aws sso login --profile dp-prod

aws ec2 describe-images --owner self --output json | jq . >tmp/prod-amis.json

# process the json files
cd get-ami-info || exit 1
go run process-ami-json.go
cd ..
