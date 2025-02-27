#!/bin/bash
podman build -t localhost/oc-daemon-test-deb12 -f test/ocserv/ocdaemon.Dockerfile .
