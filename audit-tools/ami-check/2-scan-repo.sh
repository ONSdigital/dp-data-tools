#!/usr/bin/env bash

# process repo(s) for ami id's
cd scan-repo || exit
go run process-repo.go gitdiff.go
cd ..
