#!/usr/bin/env bash

# staging - logstash 1

LOG_1="staging-logs-1.txt"

if [ -e "$LOG_1" ]; then
  echo "Removing old file $LOG_1"
  rm $LOG_1
fi

for zipped in $(find staging/logstash-1 -name logstash-plain-*.log.gz)
do
  echo "$zipped"
  zcat < "${zipped}" | grep -E 'error' >> $LOG_1
done

if [ -e "staging/logstash-1/logstash-plain.log" ]; then
  echo "staging/logstash-1/logstash-plain.log"
  cat < staging/logstash-1/logstash-plain.log | grep -E 'error' >> $LOG_1
fi

# staging - logstash 2

LOG_2="staging-logs-2.txt"

if [ -e "$LOG_2" ]; then
  echo "Removing old file $LOG_2"
  rm $LOG_2
fi

for zipped in $(find staging/logstash-2 -name logstash-plain-*.log.gz)
do
  echo "$zipped"
  zcat < "${zipped}" | grep -E 'error' >> $LOG_2
done

if [ -e "staging/logstash-2/logstash-plain.log" ]; then
  echo "staging/logstash-2/logstash-plain.log"
  cat < staging/logstash-2/logstash-plain.log | grep -E 'error' >> $LOG_2
fi

# staging - logstash 3

LOG_3="staging-logs-3.txt"

if [ -e "$LOG_3" ]; then
  echo "Removing old file $LOG_3"
  rm $LOG_3
fi

for zipped in $(find staging/logstash-3 -name logstash-plain-*.log.gz)
do
  echo "$zipped"
  zcat < "${zipped}" | grep -E 'error' >> $LOG_3
done

if [ -e "staging/logstash-3/logstash-plain.log" ]; then
  echo "staging/logstash-3/logstash-plain.log"
  cat < staging/logstash-3/logstash-plain.log | grep -E 'error' >> $LOG_3
fi
