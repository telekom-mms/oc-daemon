#!/bin/bash
docker buildx build -t oc-daemon-test-deb12 -f deb12.Dockerfile .
