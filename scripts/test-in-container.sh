#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if ! command -v docker >/dev/null 2>&1; then
  echo "error: docker is required to run tests in a container" >&2
  exit 1
fi

TEST_IMAGE=${TEST_IMAGE:-golang:1.23}
CONTAINER_NAME=${TEST_CONTAINER_NAME:-phoenix-rss-test}
DOCKER_ARGS=${DOCKER_TEST_ARGS:-}
TEST_NETWORK=${TEST_NETWORK:-phoenix-rss-net}

RAW_ARGS=("$@")
if [[ ${#RAW_ARGS[@]} -eq 0 ]]; then
  RAW_ARGS=("./...")
fi

printf -v GO_TEST_ARGS '%q ' "${RAW_ARGS[@]}"

GO_COMMAND="export PATH=\"\$PATH:/usr/local/go/bin\";"
GO_COMMAND+=" go test ${GO_TEST_ARGS}-count=1 -coverprofile=coverage.out -covermode=atomic"
GO_COMMAND+=" && go tool cover -func=coverage.out | tail -n 1"

if [[ -z "$DOCKER_ARGS" ]]; then
  if docker network inspect "$TEST_NETWORK" >/dev/null 2>&1; then
    DOCKER_ARGS="--network $TEST_NETWORK"
  else
    echo "error: docker network '$TEST_NETWORK' not found. Start docker compose or set DOCKER_TEST_ARGS/TEST_NETWORK." >&2
    exit 1
  fi
fi

EXTRA_DOCKER_ARGS=()
if [[ -n "$DOCKER_ARGS" ]]; then
  read -r -a EXTRA_DOCKER_ARGS <<<"$DOCKER_ARGS"
fi

DOCKER_CMD=(
  docker run --rm
  --name "${CONTAINER_NAME}-$$"
  -v "$ROOT_DIR":/workspace
  -w /workspace
)

if [[ -f "$ROOT_DIR/.env" ]]; then
  DOCKER_CMD+=(--env-file "$ROOT_DIR/.env")
fi

if [[ ${#EXTRA_DOCKER_ARGS[@]} -gt 0 ]]; then
  DOCKER_CMD+=("${EXTRA_DOCKER_ARGS[@]}")
fi

DOCKER_CMD+=("$TEST_IMAGE" bash -lc "$GO_COMMAND")

"${DOCKER_CMD[@]}"
