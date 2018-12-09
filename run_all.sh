#!/bin/bash
set -e
set -o pipefail

BORK_INTERFACE="${BORK_INTERFACE:-eth0}"

for DROP_PCT in 0 0.1 0.5 1 2
do
  /usr/local/google/home/jsand/coredns -dns.port 1053 -conf corefiles/google_tcp_fwd.Corefile &
  COREDNS_PID=$!
  BORK_INTERFACE=$BORK_INTERFACE BORK_IP=8.8.8.8 DROP_PCT=$DROP_PCT ./network_gremlin.sh
  sleep 3 # wait for coredns to initialize
  ./lookup --nameserver localhost --port=1053  --stats-file=$HOME/proxy_stats/drop_${DROP_PCT}.csv
  kill -15 $COREDNS_PID
  ./lookup --nameserver 8.8.8.8 --stats-file=$HOME/std_lookup/drop_${DROP_PCT}.csv
done
