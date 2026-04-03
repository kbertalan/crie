FROM golang:1.24 AS build-echo
WORKDIR /src
COPY test/echo/go.mod test/echo/go.sum ./test/echo/
RUN cd test/echo && go mod download
COPY test/echo/ ./test/echo/
RUN cd test/echo && CGO_ENABLED=0 go build -o /echo-lambda main.go

FROM golang:1.24 AS build-crie
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 go build -o /crie ./cmd/crie/crie.go

FROM gcr.io/distroless/static-debian12
COPY --from=build-echo /echo-lambda /echo-lambda
COPY --from=build-crie /crie /crie
ENV CRIE_LAMBDA_NAME=my-function
ENV CRIE_QUEUE_SIZE=10
EXPOSE 10000
ENTRYPOINT ["/crie", "/echo-lambda"]
