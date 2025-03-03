#!/bin/bash
podman build -t "localhost/oc-daemon-test-ocserv" -f "test/ocserv/ocdaemon.Dockerfile" .
