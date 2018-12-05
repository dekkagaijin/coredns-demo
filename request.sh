#!/bin/bash
endpoint="http://127.0.0.1"

while read line; do
  line_morphed="${line//./-}"
  host="$line_morphed.kubecon.critical.software"
  wget -v -O- --max-redirect=1 --header="Host: $host" $endpoint
done <data/hostnames.txt

