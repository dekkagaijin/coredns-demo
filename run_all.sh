#!/bin/bash
set -e
set -o pipefail

for DROP_PCT in 0 0.1 0.5 1 2
do
  /usr/local/google/home/jsand/coredns -dns.port 1053 -conf corefiles/google_tcp_fwd.Corefile &
  COREDNS_PID=$!
  BORK_INTERFACE=enp0s31f6 DROP_PCT=$DROP_PCT ./network_gremlin.sh
  sleep 3 # wait for coredns to initialize
  ./lookup --nameserver localhost --port=1053  --stats-file=/usr/local/google/home/jsand/proxy_stats/drop_${DROP_PCT}.csv
  kill -15 $COREDNS_PID
  sleep 2 # wait for coredns to shut down
done
