package terminator

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
)

func ReapZombies(ctx context.Context) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGCHLD)
	defer signal.Stop(sigCh)

	for {
		select {
		case <-ctx.Done():
			return
		case <-sigCh:
			for {
				var status syscall.WaitStatus
				pid, err := syscall.Wait4(-1, &status, syscall.WNOHANG, nil)
				if pid <= 0 {
					if errors.Is(err, syscall.EINTR) {
						continue
					}
					break
				}
			}
		}
	}
}
