#!/usr/bin/env bash
# Applies the OpenTofu project (builds + pushes the Go and Python lambda images,
# creates the lambdas and their public Function URLs), invokes each function with
# the test client, then destroys everything.
set -euo pipefail

cd "$(dirname "$0")"

CONCURRENCY="${1:-5}"
REPO_ROOT="$(git rev-parse --show-toplevel)"

cleanup() {
  echo "==> destroying tofu resources..."
  tofu destroy -auto-approve -input=false || true
}
trap cleanup EXIT

echo "==> tofu init..."
tofu init -input=false

echo "==> tofu apply..."
tofu apply -auto-approve -input=false

for impl in go python; do
  url="$(tofu output -raw "${impl}_function_url")"
  url="${url%/}" # the client appends the invocation path, so drop the trailing slash

  echo "==> invoking ${impl} lambda (concurrency=${CONCURRENCY}): ${url}"
  (
    cd "$REPO_ROOT"
    CRIE_ENDPOINT="$url" CLIENT_CONCURRENCY="$CONCURRENCY" go run ./test/client
  )
done

echo "==> test complete"
