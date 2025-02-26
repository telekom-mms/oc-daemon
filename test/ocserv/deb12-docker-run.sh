#!/bin/bash
cd ../..
docker run --rm --mount type=bind,src=.,dst=/oc-daemon oc-daemon-test-deb12
