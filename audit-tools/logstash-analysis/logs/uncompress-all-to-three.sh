#!/usr/bin/env bash

# logstash 1

LOG_1="logs-1.txt"

if [ -e "$LOG_1" ]; then
  echo "Removing old file $LOG_1"
  rm $LOG_1
fi

for zipped in $(find logstash-1/logstash -name logstash-plain-*.log.gz)
do
  echo "$zipped"
  zcat < "${zipped}" | grep -E 'error' >> $LOG_1
done

if [ -e "logstash-1/logstash/logstash-plain.log" ]; then
  echo "logstash-1/logstash/logstash-plain.log"
  cat < logstash-1/logstash/logstash-plain.log | grep -E 'error' >> $LOG_1
fi

# logstash 2

LOG_2="logs-2.txt"

if [ -e "$LOG_2" ]; then
  echo "Removing old file $LOG_2"
  rm $LOG_2
fi

for zipped in $(find logstash-2/logstash -name logstash-plain-*.log.gz)
do
  echo "$zipped"
  zcat < "${zipped}" | grep -E 'error' >> $LOG_2
done

if [ -e "logstash-2/logstash/logstash-plain.log" ]; then
  echo "logstash-2/logstash/logstash-plain.log"
  cat < logstash-2/logstash/logstash-plain.log | grep -E 'error' >> $LOG_2
fi

# logstash 3

LOG_3="logs-3.txt"

if [ -e "$LOG_3" ]; then
  echo "Removing old file $LOG_3"
  rm $LOG_3
fi

for zipped in $(find logstash-3/logstash -name logstash-plain-*.log.gz)
do
  echo "$zipped"
  zcat < "${zipped}" | grep -E 'error' >> $LOG_3
done

if [ -e "logstash-3/logstash/logstash-plain.log" ]; then
  echo "logstash-3/logstash/logstash-plain.log"
  cat < logstash-3/logstash/logstash-plain.log | grep -E 'error' >> $LOG_3
fi
