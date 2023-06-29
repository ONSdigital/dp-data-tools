#!/usr/bin/env bash

# sandbox - logstash 1

LOG_1="sandbox-logs-1.txt"

if [ -e "$LOG_1" ]; then
  echo "Removing old file $LOG_1"
  rm $LOG_1
fi

for zipped in $(find sandbox/logstash-1 -name logstash-plain-*.log.gz)
do
  echo "$zipped"
  zcat < "${zipped}" | grep -E 'error' >> $LOG_1
done

if [ -e "sandbox/logstash-1/logstash-plain.log" ]; then
  echo "sandbox/logstash-1/logstash-plain.log"
  cat < sandbox/logstash-1/logstash-plain.log | grep -E 'error' >> $LOG_1
fi

# sandbox - logstash 2

LOG_2="sandbox-logs-2.txt"

if [ -e "$LOG_2" ]; then
  echo "Removing old file $LOG_2"
  rm $LOG_2
fi

for zipped in $(find sandbox/logstash-2 -name logstash-plain-*.log.gz)
do
  echo "$zipped"
  zcat < "${zipped}" | grep -E 'error' >> $LOG_2
done

if [ -e "sandbox/logstash-2/logstash-plain.log" ]; then
  echo "sandbox/logstash-2/logstash-plain.log"
  cat < sandbox/logstash-2/logstash-plain.log | grep -E 'error' >> $LOG_2
fi

# sandbox - logstash 3

LOG_3="sandbox-logs-3.txt"

if [ -e "$LOG_3" ]; then
  echo "Removing old file $LOG_3"
  rm $LOG_3
fi

for zipped in $(find sandbox/logstash-3 -name logstash-plain-*.log.gz)
do
  echo "$zipped"
  zcat < "${zipped}" | grep -E 'error' >> $LOG_3
done

if [ -e "sandbox/logstash-3/logstash-plain.log" ]; then
  echo "sandbox/logstash-3/logstash-plain.log"
  cat < sandbox/logstash-3/logstash-plain.log | grep -E 'error' >> $LOG_3
fi
