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

	for range 2 {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case inv, ok := <-invocationCh:
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
		inv.ResponseCh <- invocation.Response{
			StatusCode: http.StatusInternalServerError,
			Header: http.Header{
				"content-type": []string{"application/json"},
			},
			Body:  []byte(fmt.Sprintf(`{"message": "server shutdown"}%s`, "\n")),
			Error: nil,
		}
		close(inv.ResponseCh)
	}
}
