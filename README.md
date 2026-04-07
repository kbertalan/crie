# crie

Concurrent (AWS Lambda) Runtime Interface Emulator.

Aims to be a simple tool for running your AWS Lambda Containers on your machine painlessly.

Drop in to you docker image and start your lambda code through it.

## Request Processing

crie operates in two modes depending on whether the `AWS_LAMBDA_RUNTIME_API` environment variable is set:

- **Emulate mode** (default): crie acts as a full Lambda Runtime Interface Emulator acting like AWS in local machine, managing concurrent Lambda processes and routing invocations to them.
- **Delegate mode** (`AWS_LAMBDA_RUNTIME_API` is set): crie passes through to the real AWS Lambda runtime, acting as a thin wrapper with zombie reaping.

### Emulate Mode

```
                         ┌─────────────────────────────────────────────────────────────────────────────────────┐
                         │ crie process                                                                        │
                         │                                                                                     │
 ┌────────┐  HTTP POST   │  ┌──────────────────┐  invocationCh  ┌──────────────┐                               │
 │        │ ───────────> │  │  Invoke Handler  │ ─────────────> │   Manager    │                               │
 │        │  /functions/ │  │  (HTTP handler)  │   (channel)    │ (goroutine)  │                               │
 │        │  invocations │  └──────────────────┘                └──────┬───────┘                               │
 │        │              │         │  ▲                                │                                       │
 │        │              │         │  │ responseCh                     │ try each process until one is idle    │
 │        │              │         │  │ (channel)              ┌───────┴────────────────────────────┐          │
 │ Client │              │         │  │                        │                                    │          │
 │        │              │         ▼  │                        ▼                                    ▼          │
 │        │              │  ┌──────────────────┐    ┌────────────────────────┐    ┌─────────────────────────┐  │
 │        │              │  │  Wait for        │    │   Managed Process 1    │    │   Managed Process N     │  │
 │        │              │  │  response or     │    │      (goroutine)       │    │      (goroutine)        │  │
 │        │  HTTP resp   │  │  timeout         │    │                        │    │                         │  │
 │        │ <─────────── │  └──────────────────┘    │  ┌──────────────────┐  │    │  ┌───────────────────┐  │  │
 │        │              │                          │  │  RAPI Server     │  │    │  │  RAPI Server      │  │  │
 │        │              │                          │  │  :10001          │  │    │  │  :1000N           │  │  │
 └────────┘              │                          │  │  (HTTP server)   │  │    │  │  (HTTP server)    │  │  │
                         │                          │  └──┬───────────────┘  │    │  └───────────────────┘  │  │
                         │                          │     │             ▲    │    │                         │  │
                         │                          └─────┼─────────────┼────┘    └─────────────────────────┘  │
                         │                                │             │                                      │
                         └────────────────────────────────┼─────────────┼──────────────────────────────────────┘
                                                          │             │
                                      GET /invocation/next│             │ POST /invocation/{id}/response
                                          (HTTP, blocks)  │             │ POST /invocation/{id}/error
                                                          │             │    (HTTP)
                                                          ▼             │
                                                   ┌────────────────────┴─┐
                                                   │                      │
                                                   │   Lambda Process 1   │
                                                   │   (child process)    │
                                                   │                      │
                                                   └──────────────────────┘
```

1. Client sends `POST /2015-03-31/functions/{name}/invocations` to the Invoke Handler.
2. Invoke Handler creates an `Invocation` (with a UUID and a `responseCh` channel) and sends it to `invocationCh`.
3. Manager reads from `invocationCh` and finds an idle Managed Process.
4. Managed Process starts the Lambda child process (if not already running) and passes the invocation to its RAPI Server.
5. Lambda Process calls `GET /2018-06-01/runtime/invocation/next` on the RAPI Server — this blocks until an invocation is available.
6. RAPI Server returns the request payload and Lambda-specific headers (request ID, deadline, ARN).
7. Lambda Process executes and posts the result to `POST /2018-06-01/runtime/invocation/{id}/response` (or `/error`).
8. RAPI Server writes the response to `responseCh`, unblocking the Invoke Handler.
9. Invoke Handler returns the response to the Client.

### Delegate Mode

1. crie starts the Lambda Process as a child process with the original `AWS_LAMBDA_RUNTIME_API` environment unchanged.
2. The Lambda Process communicates directly with the real AWS Lambda Runtime API — crie does not intercept or proxy these calls.
3. crie only provides zombie reaping (via SIGCHLD handling) and signal forwarding (SIGTERM/SIGINT trigger process termination).
4. When the child process exits, crie exits.
