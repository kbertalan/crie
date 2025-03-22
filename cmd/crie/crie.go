package main

import (
	"context"
	"log"

	"github.com/kbertalan/crie/internal/config"
	"github.com/kbertalan/crie/internal/process"
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
	// ctx, cancel := context.WithCancel(context.Background())
	// var wg sync.WaitGroup
	//
	// wg.Add(1)
	// go server.ListenAndServe(ctx, cfg, &wg)
	//
	// wg.Add(int(cfg.MaxConcurrency))
	// go manager.Processes(ctx, cfg, &wg)
	//
	// terminator.Wait(ctx, cancel)
	// wg.Wait()
}

func delegate(cfg config.Config) {
	ctx, cancel := context.WithCancel(context.Background())
	process.Run(ctx, cfg, cancel)
	terminator.Wait(ctx, cancel)
}
