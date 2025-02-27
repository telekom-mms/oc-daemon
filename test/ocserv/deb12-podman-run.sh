#!/bin/bash
cd ../..
podman run \
	--rm \
	--mount type=bind,src=.,dst=/oc-daemon \
	--cap-add NET_ADMIN \
	--cap-add NET_RAW \
	--device /dev/net/tun \
	--sysctl net.ipv4.conf.all.src_valid_mark=1 \
	--network oc-daemon-test \
	--name deb12 \
	localhost/oc-daemon-test-deb12
	#--privileged \
