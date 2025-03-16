#!/bin/sh

exec docker compose -f docker-compose.dev.yml up -d db

