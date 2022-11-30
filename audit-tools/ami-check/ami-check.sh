#!/usr/bin/env bash

# run all three steps

./1-get-ami-info.sh || exit 1

./2-scan-repo.sh || exit 1

./3-process-scan.sh || exit 1

