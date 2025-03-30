package main

import (
	"context"
	"errors"
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

	invocationCh := make(chan invocation.Invocation, cfg.MaxConcurrency)

	wg.Add(1)
	go server.ListenAndServe(ctx, cfg, &wg, cancel, invocationCh)

	go func() {
		for inv := range invocationCh {
			inv := inv
			go func() {
				log.Printf("[%s]: request", inv.ID)
				time.Sleep(3 * time.Second)

				inv.ResponseCh <- invocation.Response{
					StatusCode: http.StatusInternalServerError,
					Header: http.Header{
						"content-type": []string{"application/json"},
					},
					Body:  []byte(fmt.Sprintf(`{"message": "error was indeed happening for request: %s"}%s`, inv.ID, "\n")),
					Error: errors.New("error happened"),
				}
				close(inv.ResponseCh)
			}()
		}
	}()

	// wg.Add(1)
	// go manager.Processes(ctx, cfg, &wg)

	terminator.Wait(ctx, cancel)
	wg.Wait()
}

func delegate(cfg config.Config) {
	ctx, cancel := context.WithCancel(context.Background())
	process.Run(ctx, cfg, cancel)
	terminator.Wait(ctx, cancel)
}
