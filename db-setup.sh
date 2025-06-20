#!/bin/bash
set -e

BASE_PATH=$(dirname $(readlink -f ${BASH_SOURCE[0]}))

# --- Configuration ---
DB_USER="postgres"
DB_PASSWORD="password"
DB_NAME="phoenix_rss"
CONTAINER_NAME="phoenix_rss"
# This is the port used BOTH on the host AND inside the container
# because your postgresql.conf sets it to 15432.
INTERNAL_AND_EXTERNAL_PORT="15432"
DB_CONTAINER_PORT_MAPPING="5432" # The default port postgres would use without custom config
IMAGE="postgres:14"

echo "Stopping and removing old container..."
docker rm -f "$CONTAINER_NAME" > /dev/null 2>&1 || true

echo "Starting new PostgreSQL container service..."
# The port mapping should be from the host to the port Postgres is ACTUALLY listening on
docker run -d -p 127.0.0.1:$INTERNAL_AND_EXTERNAL_PORT:$INTERNAL_AND_EXTERNAL_PORT \
    --name "$CONTAINER_NAME" \
    -v "${BASE_PATH}/postgresql.conf:/etc/postgresql/postgresql.conf" \
    --tmpfs /var/lib/postgresql/data \
    -e POSTGRES_USER="$DB_USER" \
    -e POSTGRES_PASSWORD="$DB_PASSWORD" \
    "$IMAGE" \
    -c 'config_file=/etc/postgresql/postgresql.conf'

echo "Waiting for PostgreSQL to start..."
# Use the correct port (15432) that the server is actually listening on
until docker exec "$CONTAINER_NAME" pg_isready -h localhost -p "$INTERNAL_AND_EXTERNAL_PORT" -U "$DB_USER" > /dev/null 2>&1; do
  echo -n "."
  sleep 1
done
echo "PostgreSQL started!"

echo "Creating database..."
# Use the correct port (15432) here as well
docker exec -u "$DB_USER" "$CONTAINER_NAME" psql -h localhost -p "$INTERNAL_AND_EXTERNAL_PORT" -c "CREATE DATABASE \"${DB_NAME}\" OWNER=${DB_USER};"

echo "Database setup completed!"
