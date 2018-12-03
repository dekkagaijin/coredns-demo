#!/bin/bash

# See `ifconfig -a` or `ip link show` for network interfaces   
interface=eth0
ip=1.1.1.1
delay=100ms
drop_pct=50

tc qdisc del dev ${interface} root
tc qdisc add dev ${interface} root handle 1: prio
tc filter add dev ${interface} parent 1:0 protocol ip prio 1 u32 match ip dst ${ip} flowid 2:1
tc qdisc add dev ${interface} parent 1:1 handle 2: netem delay ${delay} loss ${drop_pct}%

echo "Success! Execute 'tc qdisc del dev ${interface} root' to delete rule"
