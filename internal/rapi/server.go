package rapi

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/kbertalan/crie/internal/config"
	"github.com/kbertalan/crie/internal/invocation"
	"github.com/kbertalan/crie/internal/sender"
)

type ServerConfig struct {
	Config  config.Config
	ID      string
	Address config.ListenAddress
}

type Server struct {
	mu  sync.Mutex
	cfg ServerConfig

	srv   *http.Server
	state serverState
	ch    chan struct{}
	inv   *invocation.Invocation
	errCh chan error
}

type serverState int

const (
	stopped serverState = iota
	idle
	busy
)

const (
	LambdaRuntimeAwsRequestID       = "Lambda-Runtime-Aws-Request-Id"
	LambdaRuntimeDeadlineMs         = "Lambda-Runtime-Deadline-Ms"
	LambdaRuntimeInvokedFunctionArn = "Lambda-Runtime-Invoked-Function-Arn"
	LambdaRuntimeTraceId            = "Lambda-Runtime-Trace-Id"
	LambdaRuntimeClientContext      = "Lambda-Runtime-Client-Context"
	LambdaRuntimeCognitoIdentity    = "Lambda-Runtime-Cognito-Identity"
)

func NewServer(cfg ServerConfig) *Server {
	return &Server{
		cfg:   cfg,
		ch:    make(chan struct{}, 1),
		errCh: make(chan error, 1),
	}
}

func (s *Server) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != stopped {
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /2018-06-01/runtime/invocation/next", s.serveNext)
	mux.HandleFunc("POST /2018-06-01/runtime/init/error", s.serveNext)
	mux.HandleFunc("POST /2018-06-01/runtime/invocation/{requestId}/response", s.serveInvocationResponse)
	mux.HandleFunc("POST /2018-06-01/runtime/invocation/{requestId}/error", s.serveInvocationError)

	s.srv = &http.Server{
		Addr:                         string(s.cfg.Address),
		Handler:                      mux,
		DisableGeneralOptionsHandler: true,
		ReadTimeout:                  0,
		ReadHeaderTimeout:            0,
		WriteTimeout:                 0,
		IdleTimeout:                  0,
	}

	go func() {
		err := s.srv.ListenAndServe()
		if err == nil {
			return
		}
		if errors.Is(http.ErrServerClosed, err) {
			return
		}

		log.Printf("[%s] rapi.server stopped with error: %+v", s.cfg.ID, err)
		s.mu.Lock()
		defer s.mu.Unlock()

		s.srv = nil
		s.state = stopped
		s.inv = nil
	}()

	s.state = idle
	log.Printf("[%s] rapi.server started", s.cfg.ID)
}

func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.srv == nil || s.state == stopped {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.Config.ServerShutdownTimeout)
	defer cancel()
	err := s.srv.Shutdown(ctx)
	if err != nil {
		log.Printf("[%s] rapi.server shutdown returned error: %+v", s.cfg.ID, err)
	}

	s.srv = nil
	s.state = stopped
	s.inv = nil
	close(s.ch)
	close(s.errCh)

	log.Printf("[%s] rapi.server stopped", s.cfg.ID)
}

func (s *Server) Next(inv invocation.Invocation) error {
	s.mu.Lock()
	s.state = busy
	s.inv = &inv
	s.ch <- struct{}{}
	s.mu.Unlock()

	return <-s.errCh
}

func (s *Server) serveNext(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	select {
	case <-ctx.Done():
		return
	case _, ok := <-s.ch:
		if !ok {
			sender.SendMessage(w, http.StatusNotFound, "no more invocations")
			return
		}

		target := w.Header()
		s.copyHeadersFromInvocation(target)
		s.prepareLambdaHeaders(target)

		w.WriteHeader(http.StatusOK)
		w.Write(s.inv.Request.Body)
	}
}

func (s *Server) copyHeadersFromInvocation(target http.Header) {
	if s.inv == nil {
		return
	}

	for header, values := range s.inv.Request.Header {
		target.Del(header)
		for _, value := range values {
			target.Add(header, value)
		}
	}
}

func (s *Server) prepareLambdaHeaders(target http.Header) {
	if s.inv == nil {
		return
	}

	target.Del(LambdaRuntimeAwsRequestID)
	target.Add(LambdaRuntimeAwsRequestID, s.inv.ID.String())

	target.Del(LambdaRuntimeDeadlineMs)
	target.Add(LambdaRuntimeDeadlineMs, strconv.Itoa(int(s.cfg.Config.LambdaRuntimeDeadline.Milliseconds())))

	target.Del(LambdaRuntimeInvokedFunctionArn)
	target.Add(LambdaRuntimeInvokedFunctionArn, s.cfg.Config.LambdaRuntimeInvokedFunctionArn)

	target.Del(LambdaRuntimeTraceId)
	// TODO set trace id

	target.Del(LambdaRuntimeClientContext)
	// TODO set client context

	target.Del(LambdaRuntimeCognitoIdentity)
	// TODO set cognito identity
}

func (s *Server) serveInitializationError(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	log.Printf("initialization error: %s", string(body))
	w.WriteHeader(http.StatusOK)
}

func (s *Server) serveInvocationError(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	log.Printf("server invocation error: %s", string(body))
	w.WriteHeader(http.StatusOK)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.state = idle
	s.inv = nil
	s.errCh <- errors.New(string(body))
}

func (s *Server) serveInvocationResponse(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	log.Printf("server invocation response: %s", string(body))
	w.WriteHeader(http.StatusOK)

	s.inv.ResponseCh <- invocation.Response{
		StatusCode: http.StatusOK,
		Header:     nil,
		Body:       body,
		Error:      nil,
	}
	close(s.inv.ResponseCh)
	s.errCh <- nil

	s.mu.Lock()
	defer s.mu.Unlock()

	s.state = idle
	s.inv = nil
}
