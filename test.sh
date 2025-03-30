#!/usr/bin/env bash

go build -o crie ./cmd/crie/crie.go

export CRIE_LAMBDA_NAME='my-function'
./crie sh -c 'echo "dummy command"' &
pid=$!

sleep 0.5

function call() {
  curl http://localhost:10000/2015-03-31/functions/${CRIE_LAMBDA_NAME}/invocations -d '{}'
}

call &
call &
call

echo
kill $pid
