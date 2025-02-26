#!/bin/bash
docker run --rm --mount type=bind,src=.,dst=/ocserv -P oc-daemon-test-ocserv
