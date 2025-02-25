#!/bin/bash

COMPOSE=podman-compose

$COMPOSE --file "$PWD/test/ocserv/podman/compose.yml" down
