#!/usr/bin/env bash

# process repo(s) for ami id's
cd process-scan || exit
go run process-scan.go
cd ..
