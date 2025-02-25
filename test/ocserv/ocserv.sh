#!/bin/bash
set -e

# create ca key, ca certificate
certtool --generate-privkey --outfile /etc/ocserv/ca-key.pem && \
certtool --generate-self-signed \
	--load-privkey /etc/ocserv/ca-key.pem \
	--template /ocserv/ca.tmpl \
	--outfile /etc/ocserv/ca-cert.pem

# create server key, server certificate
certtool --generate-privkey --outfile /etc/ocserv/server-key.pem && \
certtool --generate-certificate \
	--load-privkey /etc/ocserv/server-key.pem \
	--load-ca-certificate /etc/ocserv/ca-cert.pem \
	--load-ca-privkey /etc/ocserv/ca-key.pem \
	--template /ocserv/server.tmpl \
	--outfile /etc/ocserv/server-cert.pem

# add test user
echo test_password | ocpasswd -c /etc/ocserv/passwd test_user

# create tun device
mkdir -p /dev/net
mknod /dev/net/tun c 10 200

# start ocserv
exec ocserv -f -c /ocserv/ocserv.conf
