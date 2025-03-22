package process

import (
	"context"
	"log"
	"os"
	"os/exec"

	"github.com/kbertalan/crie/internal/config"
)

func Run(ctx context.Context, cfg config.Config, cancel context.CancelFunc) {
	cmd := exec.Command(cfg.CommandName, cfg.CommandArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = cfg.OriginalEnvironment

	go func() {
		defer cancel()
		err := cmd.Run()
		if err != nil {
			log.Printf("command terminated with error: %+v", err)
		}
	}()

	go func() {
		<-ctx.Done()

		if cmd.Process == nil {
			return
		}

		if cmd.ProcessState != nil {
			return
		}

		if err := cmd.Process.Kill(); err != nil {
			log.Printf("cannot kill to process: %+v", err)
		}
	}()
}
