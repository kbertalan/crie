package process

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/kbertalan/crie/internal/config"
)

func Delegate(ctx context.Context, cfg config.Config, cancel context.CancelFunc) {
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

func Start(ctx context.Context, cfg config.Config, rapi config.ListenAddress) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, cfg.CommandName, cfg.CommandArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = append(cmd.Env, cfg.OriginalEnvironment...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_LAMBDA_RUNTIME_API=%s", rapi.AwsLambdaRuntimeAPI()))

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}
