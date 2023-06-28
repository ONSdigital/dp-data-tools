#!/usr/local/bin/bash

dp scp prod logstash 1 --pull --recurse /var/log/logstash/ logstash-1/

dp scp prod logstash 2 --pull --recurse /var/log/logstash/ logstash-2/

dp scp prod logstash 3 --pull --recurse /var/log/logstash/ logstash-3/
