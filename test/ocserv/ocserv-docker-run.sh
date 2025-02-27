#!/bin/bash
docker run \
	--rm \
	--mount type=bind,src=.,dst=/ocserv \
	-P \
	--cap-add NET_ADMIN \
	--network oc-daemon-test \
	--name ocserv \
	oc-daemon-test-ocserv
