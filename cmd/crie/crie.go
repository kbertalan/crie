package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/kbertalan/crie/internal/config"
	"github.com/kbertalan/crie/internal/invocation"
	"github.com/kbertalan/crie/internal/manager"
	"github.com/kbertalan/crie/internal/process"
	"github.com/kbertalan/crie/internal/server"
	"github.com/kbertalan/crie/internal/terminator"
)

func main() {
	cfg, err := config.Detect()
	if err != nil {
		log.Fatalf("configuration error: %+v", err)
	}

	if cfg.OriginalAWSLambdaRuntimeAPI != "" {
		delegate(cfg)
	} else {
		emulate(cfg)
	}
}

func emulate(cfg config.Config) {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	invocationCh := make(chan invocation.Invocation, cfg.QueueSize)

	wg.Add(1)
	go server.ListenAndServe(ctx, cfg, &wg, cancel, invocationCh)

	processCfgs := make([]manager.ProcessConfig, cfg.MaxConcurrency)
	for i := range cfg.MaxConcurrency {
		processCfgs[i] = manager.ProcessConfig{
			ID:    fmt.Sprintf("pid-%d", i+1),
			Start: i < cfg.InitialConcurrency,
		}
	}

	wg.Add(1)
	go manager.Processes(ctx, cfg, processCfgs, invocationCh, &wg)

	terminator.Wait(ctx, cancel)
	log.Println("shutting down started")

	go cleanupPendingInvocations(invocationCh)

	wg.Wait()
	log.Println("shutting down completed")
}

func delegate(cfg config.Config) {
	ctx, cancel := context.WithCancel(context.Background())
	process.Run(ctx, cfg, cancel)
	terminator.Wait(ctx, cancel)
}

func cleanupPendingInvocations(invocationCh <-chan invocation.Invocation) {
	for inv := range invocationCh {
		inv.ResponseCh <- invocation.ResponseMessage(http.StatusInternalServerError, "server shutdown")
		close(inv.ResponseCh)
	}
}
