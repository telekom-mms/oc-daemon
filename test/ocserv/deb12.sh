#!/bin/bash
set -e

# install oc-daemon
apt update
apt install -y procps systemd-resolved
apt install -y /oc-daemon/dist/*.deb

# create tun device
mkdir -p /dev/net
mknod /dev/net/tun c 10 200

# start dbus
dbus-daemon --config-file=/usr/share/dbus-1/system.conf --print-address

# start ocserv
exec oc-daemon
