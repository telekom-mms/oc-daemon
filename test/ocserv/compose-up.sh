#!/bin/bash

COMPOSE=podman-compose

COMPOSE_PARALLEL_LIMIT=1 $COMPOSE \
	--file "$PWD/test/ocserv/podman/compose.yml" \
	up --build --detach
