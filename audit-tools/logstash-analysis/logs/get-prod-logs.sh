#!/usr/local/bin/bash

if [ -d "prod" ]; then rm -Rf prod; fi

dp scp prod logstash 1 --pull --recurse /var/log/logstash/ logstash-1/

dp scp prod logstash 2 --pull --recurse /var/log/logstash/ logstash-2/

dp scp prod logstash 3 --pull --recurse /var/log/logstash/ logstash-3/

mkdir -p prod

mv logstash-1 prod/logstash-1

mv logstash-2 prod/logstash-2

mv logstash-3 prod/logstash-3
