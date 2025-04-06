#!/bin/sh

COMPOSE_PROVIDER="${COMPOSE_PROVIDER:-docker compose}"

SCRIPT_PATH="$(realpath "${BASH_SOURCE[0]}")"
SCRIPT_DIR="$(dirname "$SCRIPT_PATH")"
PROJECT_ROOT="$(realpath "$SCRIPT_DIR/..")"

cd ${PROJECT_ROOT}

${COMPOSE_PROVIDER} \
    -f ${PROJECT_ROOT}/docker/docker-compose.base.yml \
    exec db sh

