#!/bin/bash
#
# This script is a podman version of the goreleaser build script "build.sh"
# that uses an alternative goreleaser configuration that enables the race
# detector and coverage in the binaries.
# It should be run from the root directory of the git repository.

# set target directory in container
TARGET=/code

# run container
podman run \
	--rm \
	--mount type=bind,source="$PWD",target="$TARGET" \
	--workdir "$TARGET" \
	--env GOCACHE=/tmp \
	docker.io/goreleaser/goreleaser \
	release \
	--snapshot \
	--clean \
	--config "test/ocserv/oc-daemon/goreleaser.yml"
