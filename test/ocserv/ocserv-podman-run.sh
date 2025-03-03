#!/bin/bash
podman run \
	--rm \
	--mount type=bind,src=.,dst=/ocserv \
	-P \
	--cap-add NET_ADMIN \
	--device /dev/net/tun \
	--network oc-daemon-test \
	--name ocserv \
	localhost/oc-daemon-test-ocserv:latest
