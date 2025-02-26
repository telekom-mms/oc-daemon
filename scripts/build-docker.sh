#!/bin/bash
#
# This script is a docker version of the goreleaser build script "build.sh".
# It should be run from the root directory of the git repository.

# set target directory in container, user ID, group ID
TARGET=/code
USER=$(id -u)
GROUP=$(id -g)

# run container
docker run \
	--rm \
	--mount type=bind,source="$PWD",target="$TARGET" \
	--workdir "$TARGET" \
	--user "$USER:$GROUP" \
	--env GOCACHE=/tmp \
	goreleaser/goreleaser \
	release \
	--snapshot \
	--clean
