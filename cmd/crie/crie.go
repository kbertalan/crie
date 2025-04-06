package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/kbertalan/crie/internal/config"
	"github.com/kbertalan/crie/internal/invocation"
	"github.com/kbertalan/crie/internal/process"
	"github.com/kbertalan/crie/internal/queue"
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

	invocationCh := make(chan invocation.Invocation)

	wg.Add(1)
	go server.ListenAndServe(ctx, cfg, &wg, cancel, invocationCh)

	queuedInvocationCh := queue.Start(ctx, cfg, invocationCh)

	for range 2 {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case inv, ok := <-queuedInvocationCh:
					if !ok {
						return
					}
					log.Printf("[%s]: request", inv.ID)
					time.Sleep(3 * time.Second)

					inv.ResponseCh <- invocation.Response{
						StatusCode: http.StatusOK,
						Header: http.Header{
							"content-type": []string{"application/json"},
						},
						Body:  []byte(fmt.Sprintf(`{"message": "ok: %s"}%s`, inv.ID, "\n")),
						Error: nil,
					}
					close(inv.ResponseCh)
				}
			}
		}()
	}

	// wg.Add(1)
	// go manager.Processes(ctx, cfg, &wg)

	terminator.Wait(ctx, cancel)
	log.Println("shutting down started")
	wg.Wait()
	log.Println("shutting down completed")
}

func delegate(cfg config.Config) {
	ctx, cancel := context.WithCancel(context.Background())
	process.Run(ctx, cfg, cancel)
	terminator.Wait(ctx, cancel)
}
