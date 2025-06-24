#!/bin/bash
set -e

BASE_PATH=$(dirname $(readlink -f ${BASH_SOURCE[0]}))

CONTAINER_NAME="phoenix_redis"
IMAGE="redis:7-alpine"
HOST_PORT="6379"
CONTAINER_PORT="6379"

echo "Stopping and removing old container..."
docker rm -f "$CONTAINER_NAME" > /dev/null 2>&1 || true

echo "Starting new Redis container service..."
docker run -d -p 127.0.0.1:$HOST_PORT:$CONTAINER_PORT \
    --name "$CONTAINER_NAME" \
    "$IMAGE"

echo "Waiting for Redis to start..."
until docker exec "$CONTAINER_NAME" redis-cli ping | grep -q "PONG"; do
  echo -n "."
  sleep 1
done

echo "Redis started!"
