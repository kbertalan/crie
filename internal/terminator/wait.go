package terminator

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func Wait(ctx context.Context, cancel context.CancelFunc) {
	defer cancel()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGINT)

	select {
	case <-signalCh:
		return
	case <-ctx.Done():
		return
	}
}
