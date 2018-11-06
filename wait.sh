#!/bin/bash

httpUrl="http://localhost:8065"

while true; do
  status_code=$(curl --write-out %{http_code} --silent --output /dev/null "$httpUrl")

  if [ $status_code -ne 000 ]; then
    break
  else
    $(sleep 2)
  fi
done
