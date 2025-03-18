#!/bin/bash

ip=$1
host=$2

echo "$ip $host" >>/etc/hosts
