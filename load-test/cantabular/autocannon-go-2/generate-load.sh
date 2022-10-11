#!/usr/bin/env bash

set -e

mkdir -p "$HOME/tmp"

log=$HOME/tmp/generate-load-$(date '+%Y-%m-%d-%T').log

# time in GMT
expected_time=$1

run(){
    echo expected_time="$expected_time"

    while [[ "$(date '+%H:%M')" != "$expected_time" ]]; do
        sleep 0.2
    done

    ./generate-load
}

# Run application in background so that we can return immediately to allow makefile to deploy more
# instances of the application on other boxes that will then all be run to start at the same time.
run > "$log" 2>&1 &
