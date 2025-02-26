#!/bin/bash
docker buildx build -t oc-daemon-test-ocserv -f ocserv.Dockerfile .
