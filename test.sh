#!/usr/bin/env bash

(cd ./test/echo; go build -o ../../echo-lambda main.go)
go build -o crie ./cmd/crie/crie.go

export CRIE_LAMBDA_NAME='my-function'
export CRIE_QUEUE_SIZE=2
./crie ./echo-lambda &
pid=$!

sleep 0.5

function call() {
  # curl -s http://localhost:10000/2015-03-31/functions/${CRIE_LAMBDA_NAME}/invocations -d '{"call": "true"}' > /dev/null
  AWS_DEFAULT_REGION=us-east-1 \
  AWS_ACCESS_KEY_ID="id" \
  AWS_SECRET_ACCESS_KEY="key" \
  aws lambda invoke \
      --function-name ${CRIE_LAMBDA_NAME} \
      --endpoint-url http://localhost:10000 \
      --cli-binary-format raw-in-base64-out \
      --output json \
      --payload '{"key": "value"}' \
      /dev/null
      # --debug \
}

for i in $(seq 1 5); do
  call &
done

sleep 1

kill $pid
wait %1
