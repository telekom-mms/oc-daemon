#!/bin/bash
#podman build -t localhost/oc-daemon-test-deb12 -f test/ocserv/ocdaemon.Dockerfile .
podman image build --rm --target oc-daemon -t oc-daemon-test-deb12 -f test/ocserv/both.Dockerfile .
