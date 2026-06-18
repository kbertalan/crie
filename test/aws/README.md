# AWS test deployment

A small OpenTofu project that deploys the Go and Python test handlers to AWS
Lambda as container images, each wrapped by `crie` running in **delegate mode**
(AWS sets `AWS_LAMBDA_RUNTIME_API`, so crie acts as a thin wrapper with zombie
reaping). This exercises crie on real Lambda using the exact images from
`test/go` and `test/python`.

## What it creates

- An ECR repository per lambda (`crie-test-go`, `crie-test-python`).
- The container images, built from the repo root and pushed to ECR by the
  `kreuzwerker/docker` provider (so a single `tofu apply` builds + deploys).
- A shared Lambda execution role (CloudWatch logs).
- Two `x86_64` Lambda functions, each with a public **Function URL**
  (`authorization_type = NONE`) so the unsigned `test/client` can invoke them.

## Prerequisites

- AWS credentials in the environment (e.g. `AWS_PROFILE` / `AWS_REGION`).
- A running Docker daemon. Images are built for `linux/amd64`; on an arm64 host
  you need buildx/qemu emulation.
- `tofu` and the Go toolchain (the test client runs via `go run`).

## Usage

```sh
./test.sh            # apply, invoke both lambdas, then destroy
./test.sh 20         # same, with client concurrency 20
```

Or drive tofu directly:

```sh
tofu init
tofu apply
tofu output function_urls
tofu destroy
```

The default region is `eu-central-1`; override with `-var region=...`.

> The client posts to the Function URL, so each lambda receives the Function-URL
> request event and echoes it back (HTTP 200) — a smoke test of crie's delegate
> mode, not a raw-payload round trip.
