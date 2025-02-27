#!/bin/bash
cd ../..
docker run \
	--rm \
	--mount type=bind,src=.,dst=/oc-daemon \
	--cap-add NET_ADMIN \
	--network oc-daemon-test \
	--name deb12 \
	oc-daemon-test-deb12
