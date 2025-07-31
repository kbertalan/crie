#!/usr/bin/env bash

(cd ./test/echo; go build -o ../../echo-lambda main.go)
go build -o crie ./cmd/crie/crie.go

export CRIE_LAMBDA_NAME='my-function'
export CRIE_QUEUE_SIZE=2
./crie ./echo-lambda &
pid=$!

sleep 0.5

function call() {
  curl -s http://localhost:10000/2015-03-31/functions/${CRIE_LAMBDA_NAME}/invocations -d '"call"' > /dev/null
}

call &
call &
call &
call &
call &

sleep 1

kill $pid
wait %1
