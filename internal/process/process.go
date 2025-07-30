package process

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"

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

type Process struct {
	id       string
	cfg      config.Config
	rapi     config.ListenAddress
	cmd      *exec.Cmd
	stopping bool
}

func NewProcess(id string, cfg config.Config, rapi config.ListenAddress) *Process {
	return &Process{
		id:   id,
		cfg:  cfg,
		rapi: rapi,
		cmd:  nil,
	}
}

func (p *Process) Start() error {
	p.stopping = false

	if p.cmd != nil && p.cmd.ProcessState == nil {
		return nil
	}

	p.cmd = exec.Command(p.cfg.CommandName, p.cfg.CommandArgs...)
	p.cmd.Stdin = os.Stdin
	p.cmd.Stdout = os.Stdout
	p.cmd.Stderr = os.Stderr

	p.cmd.Env = append(p.cmd.Env, p.cfg.OriginalEnvironment...)
	p.cmd.Env = append(p.cmd.Env, fmt.Sprintf("AWS_LAMBDA_RUNTIME_API=%s", p.rapi.AwsLambdaRuntimeAPI()))

	if err := p.cmd.Start(); err != nil {
		log.Printf("[%s] process start failed: %+v", p.id, err)
		return err
	}

	log.Printf("[%s] process started", p.id)

	go func() {
		p.cmd.Wait()
		log.Printf("[%s] process ended", p.id)
		if !p.stopping {
			p.Start()
		}
	}()

	return nil
}

func (p *Process) Stop() {
	if p.cmd == nil || p.cmd.ProcessState != nil {
		return
	}

	p.stopping = true

	if p.cmd.Process == nil {
		return
	}

	if err := p.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		log.Printf("[%s] process ended with error: %+v", p.id, err)
		return
	}
}
