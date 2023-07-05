#!/usr/local/bin/bash

if [ -d "staging" ]; then rm -Rf staging; fi

dp scp staging logstash 1 --pull --recurse /var/log/logstash/ logstash-1/

dp scp staging logstash 2 --pull --recurse /var/log/logstash/ logstash-2/

dp scp staging logstash 3 --pull --recurse /var/log/logstash/ logstash-3/

mkdir -p staging

mv logstash-1 staging/logstash-1

mv logstash-2 staging/logstash-2

mv logstash-3 staging/logstash-3
