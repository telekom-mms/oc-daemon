#!/bin/bash

COMPOSE_PARALLEL_LIMIT=1 podman-compose --file "$PWD/test/ocserv/compose.yml" up --detach
