#!/bin/bash

# See `ifconfig -a` or `ip link show` for network interfaces
BORK_INTERFACE="${BORK_INTERFACE:-eth0}"
BORK_IP="${BORK_IP:-1.1.1.1}"
DELAY="${DELAY:-100ms}"
DROP_PCT="${DROP_PCT:-33}"

tc qdisc del dev ${BORK_INTERFACE} root &> /dev/null
tc qdisc add dev ${BORK_INTERFACE} root handle 1: prio
tc filter add dev ${BORK_INTERFACE} parent 1:0 protocol ip prio 1 u32 match ip dst ${BORK_IP} flowid 2:1
tc qdisc add dev ${BORK_INTERFACE} parent 1:1 handle 2: netem delay ${DELAY} loss ${DROP_PCT}%

echo "Success! Execute 'tc qdisc del dev ${BORK_INTERFACE} root' to delete rule"