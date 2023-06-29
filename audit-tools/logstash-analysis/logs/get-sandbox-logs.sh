#!/usr/local/bin/bash

dp scp sandbox logstash 1 --pull --recurse /var/log/logstash/ logstash-1/

dp scp sandbox logstash 2 --pull --recurse /var/log/logstash/ logstash-2/

dp scp sandbox logstash 3 --pull --recurse /var/log/logstash/ logstash-3/

mkdir -p sandbox

mv logstash-1 sandbox/logstash-1

mv logstash-2 sandbox/logstash-2

mv logstash-3 sandbox/logstash-3
