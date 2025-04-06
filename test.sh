#!/usr/bin/env bash

go build -o crie ./cmd/crie/crie.go

export CRIE_LAMBDA_NAME='my-function'
export CRIE_QUEUE_SIZE=2
./crie sh -c 'echo "dummy command"' &
pid=$!

sleep 0.5

function call() {
  curl http://localhost:10000/2015-03-31/functions/${CRIE_LAMBDA_NAME}/invocations -d '{}'
}

call &
call &
call &
call &
call &

sleep 1

echo
kill $pid

sleep 12
