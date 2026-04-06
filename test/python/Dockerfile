FROM golang:1.24 AS build-crie

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 go build -o /crie ./cmd/crie/crie.go


FROM python:3.13-slim

WORKDIR /app
RUN pip install awslambdaric
COPY test/python/main.py .

COPY --from=build-crie /crie /crie
ENV CRIE_LAMBDA_NAME=my-function
ENV CRIE_QUEUE_SIZE=10
EXPOSE 10000
ENTRYPOINT ["/crie", "python", "-m", "awslambdaric"]
CMD ["main.handler"]
