FROM golang:1.24 AS build
WORKDIR /src
COPY test/client/main.go ./
RUN CGO_ENABLED=0 go build -o /client main.go

FROM gcr.io/distroless/static-debian12
COPY --from=build /client /client
ENV CRIE_ENDPOINT=http://crie:10000
ENV CRIE_LAMBDA_NAME=my-function
ENV CLIENT_CONCURRENCY=5
ENTRYPOINT ["/client"]
