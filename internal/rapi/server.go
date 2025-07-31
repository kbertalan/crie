package rapi

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/kbertalan/crie/internal/config"
	"github.com/kbertalan/crie/internal/invocation"
	"github.com/kbertalan/crie/internal/sender"
)

type Server struct {
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc

	id   string
	cfg  config.Config
	rapi config.ListenAddress

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

func NewServer(id string, cfg config.Config, rapi config.ListenAddress) *Server {
	return &Server{
		id:    id,
		cfg:   cfg,
		rapi:  rapi,
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
	mux.HandleFunc("POST /2018-06-01/runtime/init/error", s.serveInitializationError)
	mux.HandleFunc("POST /2018-06-01/runtime/invocation/{requestId}/response", s.serveInvocationResponse)
	mux.HandleFunc("POST /2018-06-01/runtime/invocation/{requestId}/error", s.serveInvocationError)

	s.srv = &http.Server{
		Addr:                         string(s.rapi),
		Handler:                      mux,
		DisableGeneralOptionsHandler: true,
		ReadTimeout:                  0,
		ReadHeaderTimeout:            0,
		WriteTimeout:                 0,
		IdleTimeout:                  0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx
	s.cancel = cancel

	go func() {
		err := s.srv.ListenAndServe()
		if err == nil {
			return
		}
		if errors.Is(http.ErrServerClosed, err) {
			return
		}

		log.Printf("[%s] rapi.server stopped with error: %+v", s.id, err)
		s.mu.Lock()
		defer s.mu.Unlock()

		s.cancel()
		s.sendInvocationError(http.StatusInternalServerError, "unknown error")

		s.srv = nil
		s.state = stopped
		s.inv = nil
	}()

	s.state = idle
	log.Printf("[%s] rapi.server started", s.id)
}

func (s *Server) Stop() {
	s.mu.Lock()

	if s.srv == nil || s.state == stopped {
		return
	}

	s.cancel()
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.ServerShutdownTimeout)
	defer cancel()
	err := s.srv.Shutdown(ctx)
	if err != nil {
		log.Printf("[%s] rapi.server shutdown returned error: %+v", s.id, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.inv != nil {
		log.Printf("[%s] rapi.server had a pending invocation [%s], sending error", s.id, s.inv.ID)
		s.sendInvocationError(http.StatusInternalServerError, "server shutdown")
	}

	s.srv = nil
	s.state = stopped
	s.inv = nil
	close(s.ch)
	close(s.errCh)

	log.Printf("[%s] rapi.server stopped", s.id)
}

func (s *Server) Next(inv invocation.Invocation) error {
	s.mu.Lock()
	s.state = busy
	s.inv = &inv
	s.ch <- struct{}{}
	s.mu.Unlock()

	return <-s.errCh
}

func (s *Server) sendInvocationError(status int, format string, args ...any) {
	if s.inv == nil {
		return
	}

	s.inv.ResponseCh <- invocation.ResponseMessage(status, format, args...)
	close(s.inv.ResponseCh)
}

func (s *Server) serveNext(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	select {
	case <-ctx.Done():
		return
	case <-s.ctx.Done():
		w.WriteHeader(http.StatusInternalServerError)
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
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		log.Printf("[%s] sent next request [%s]", s.id, s.inv.ID)
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
	target.Add(LambdaRuntimeDeadlineMs, strconv.FormatInt(time.Now().Add(s.cfg.LambdaRuntimeDeadline).UnixMilli(), 10))

	target.Del(LambdaRuntimeInvokedFunctionArn)
	target.Add(LambdaRuntimeInvokedFunctionArn, s.cfg.LambdaRuntimeInvokedFunctionArn)

	target.Del(LambdaRuntimeTraceId)
	// TODO set trace id

	target.Del(LambdaRuntimeClientContext)
	// TODO set client context

	target.Del(LambdaRuntimeCognitoIdentity)
	// TODO set cognito identity
}

func (s *Server) serveInitializationError(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	log.Printf("[%s] initialization error: %s", s.id, string(body))
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) serveInvocationError(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	log.Printf("[%s] server invocation error [%s]: %s", s.id, s.inv.ID, string(body))
	w.WriteHeader(http.StatusAccepted)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.state = idle
	s.inv = nil
	s.errCh <- errors.New(string(body))
}

func (s *Server) serveInvocationResponse(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	log.Printf("[%s] server invocation response [%s]: %s", s.id, s.inv.ID, string(body))
	w.WriteHeader(http.StatusAccepted)

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
