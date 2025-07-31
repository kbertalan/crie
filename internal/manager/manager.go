package manager

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/kbertalan/crie/internal/config"
	"github.com/kbertalan/crie/internal/invocation"
	"github.com/kbertalan/crie/internal/process"
	"github.com/kbertalan/crie/internal/rapi"
)

type ProcessConfig struct {
	ID    string
	Start bool
}

type mgr struct {
	cfg       config.Config
	ch        <-chan invocation.Invocation
	processes []*managedProcess
}

func Processes(ctx context.Context, cfg config.Config, processCfgs []ProcessConfig, invocationCh <-chan invocation.Invocation, wg *sync.WaitGroup) {
	defer wg.Done()

	processes := make([]*managedProcess, 0, len(processCfgs))
	for i, processCfg := range processCfgs {
		address := cfg.ServerAddress.ProcessAddress(i)
		p := managedProcess{
			id: processCfg.ID,
			rapi: rapi.NewServer(rapi.ServerConfig{
				Config:  cfg,
				ID:      processCfg.ID,
				Address: address,
			}),
			proc: process.NewProcess(processCfg.ID, cfg, address),
		}

		if processCfg.Start {
			p.Start()
		}

		processes = append(processes, &p)
	}

	m := mgr{
		cfg:       cfg,
		ch:        invocationCh,
		processes: processes,
	}

	m.run(ctx)
}

func (m *mgr) run(ctx context.Context) {
	defer m.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case inv, ok := <-m.ch:
			if !ok {
				return
			}
			log.Printf("[%s]: request", inv.ID)
			m.handle(ctx, inv)
		}
	}
}

func (m *mgr) handle(ctx context.Context, inv invocation.Invocation) {
	for attempt := range m.cfg.MaxHandleAttempts {
		if attempt > 0 {
			time.Sleep(m.cfg.DelayBetweenHandleAttempts)
		}
		for _, p := range m.processes {
			select {
			case <-ctx.Done():
				inv.ResponseCh <- invocation.ResponseMessage(http.StatusInternalServerError, "server shutdown")
				close(inv.ResponseCh)
				return
			default:
				if ok := p.TryHandle(ctx, inv); ok {
					return
				}
			}
		}
	}

	inv.ResponseCh <- invocation.ResponseMessage(http.StatusGatewayTimeout, "could not find suitable backend for invocation: %s", inv.ID)
	close(inv.ResponseCh)
}

func (m *mgr) Close() {
	for _, p := range m.processes {
		p.Stop()
	}
}

type managedProcess struct {
	mu     sync.Mutex
	id     string
	rapi   *rapi.Server
	proc   *process.Process
	status managedProcessStatus
}

type managedProcessStatus int

const (
	idle managedProcessStatus = iota
	processing
)

func (p *managedProcess) Start() {
	log.Printf("[%s] starting", p.id)
	p.rapi.Start()
	p.proc.Start()
}

func (p *managedProcess) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.proc.Stop()
	p.rapi.Stop()
}

func (p *managedProcess) TryHandle(ctx context.Context, inv invocation.Invocation) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.status == processing {
		return false
	}

	p.status = processing

	go func() {
		p.rapi.Start()
		p.proc.Start()
		err := p.rapi.Next(inv)
		if err != nil {
			log.Printf("[%s] invocation [%s] returned error: %+v", p.id, inv.ID, err)
		}

		p.mu.Lock()
		defer p.mu.Unlock()
		p.status = idle
	}()
	return true
}
