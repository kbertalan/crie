#!/usr/bin/env bash
set -euo pipefail

CONCURRENCY="${1:-5}"
NETWORK="crie-test"

cleanup() {
  echo "cleaning up..."
  docker rm -f crie-server 2>/dev/null || true
  docker network rm "$NETWORK" 2>/dev/null || true
}
trap cleanup EXIT

echo "building crie server image..."
docker build -t crie-server -f test/echo/Dockerfile .

echo "building test client image..."
docker build -t crie-client -f test/client/Dockerfile .

docker network create "$NETWORK" 2>/dev/null || true

echo "starting crie server..."
docker run -d --rm --name crie-server --network "$NETWORK" --network-alias crie crie-server

echo "waiting for crie server to be ready..."
sleep 1

echo "running test client with concurrency=${CONCURRENCY}..."
docker run --rm --network "$NETWORK" -e "CLIENT_CONCURRENCY=${CONCURRENCY}" crie-client

echo "test complete"
