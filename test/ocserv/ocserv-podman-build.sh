#!/bin/bash
#podman build -t "localhost/oc-daemon-test-ocserv" -f "test/ocserv/ocserv.Dockerfile" .
podman image build --rm --target ocserv -t oc-daemon-test-ocserv -f test/ocserv/both.Dockerfile .
