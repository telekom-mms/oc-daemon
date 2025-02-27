#!/bin/bash
set -e

# install oc-daemon
apt update
apt install -y procps systemd-resolved gnutls-bin
apt install -y /oc-daemon/dist/*.deb

# create client key, client certificate
certtool --generate-privkey --outfile /key.pem
certtool --generate-self-signed \
	--load-privkey /key.pem \
	--template /oc-daemon/test/ocserv/client.tmpl \
	--outfile /cert.pem

# create tun device
mkdir -p /dev/net
mknod /dev/net/tun c 10 200

# start dbus
dbus-daemon --config-file=/usr/share/dbus-1/system.conf --print-address

# start ocserv
exec oc-daemon
