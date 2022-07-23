#!/bin/bash

GIT_TAG=$(git describe --tags --abbrev=0)
GIT_COMMIT=$(git rev-list -1 HEAD)
GIT_VERSION="$GIT_TAG-$GIT_COMMIT"

VERSION="github.com/T-Systems-MMS/oc-daemon/internal/daemon.Version=$GIT_VERSION"

go build -ldflags "-X $VERSION" ./cmd/oc-daemon
go build -ldflags "-X $VERSION" ./cmd/oc-daemon-vpncscript
go build -ldflags "-X $VERSION" ./cmd/oc-client
