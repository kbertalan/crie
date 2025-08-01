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

	srv       *http.Server
	state     serverState
	inv       *invocation.Invocation
	lastNext  time.Time
	lastStart time.Time
	nextCh    chan struct{}
	doneCh    chan struct{}
}

type serverState int

const (
	stopped serverState = iota
	initializing
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
	ContentType                     = "Content-Type"
	ContentTypeApplicationJSON      = "application/json"
)

func NewServer(id string, cfg config.Config, rapi config.ListenAddress) *Server {
	return &Server{
		id:     id,
		cfg:    cfg,
		rapi:   rapi,
		nextCh: make(chan struct{}, 1),
		doneCh: make(chan struct{}, 1),
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
	s.lastStart = time.Now()

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

	s.state = initializing
	log.Printf("[%s] rapi.server started", s.id)
}

func (s *Server) Stop() {
	s.mu.Lock()

	if s.srv == nil || s.state == stopped {
		s.mu.Unlock()
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
	close(s.nextCh)
	close(s.doneCh)

	log.Printf("[%s] rapi.server stopped", s.id)
}

func (s *Server) Next(inv invocation.Invocation) {
	s.mu.Lock()
	s.inv = &inv
	s.nextCh <- struct{}{}
	s.mu.Unlock()

	<-s.doneCh
}

func (s *Server) sendInvocationError(status int, format string, args ...any) {
	if s.inv == nil {
		return
	}

	s.inv.ResponseCh <- invocation.ResponseMessage(status, format, args...)
	close(s.inv.ResponseCh)
}

func (s *Server) serveNext(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	if s.state == initializing {
		log.Printf("[%s] initialization took %s", s.id, time.Since(s.lastStart))
		s.state = idle
	}
	s.mu.Unlock()

	ctx := r.Context()
	select {
	case <-ctx.Done():
		return
	case <-s.ctx.Done():
		w.WriteHeader(http.StatusInternalServerError)
		return
	case _, ok := <-s.nextCh:
		if !ok {
			sender.SendMessage(w, http.StatusNotFound, "no more invocations")
			return
		}

		s.mu.Lock()
		s.state = busy
		s.lastNext = time.Now()
		s.mu.Unlock()

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

	if target.Get(ContentType) == "" {
		target.Add(ContentType, ContentTypeApplicationJSON)
	}
}

func (s *Server) serveInitializationError(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[%s] could not read initialization error response: %+v", s.id, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf("[%s] initialization error: %s", s.id, string(body))
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) serveInvocationError(w http.ResponseWriter, r *http.Request) {
	if body, err := io.ReadAll(r.Body); err == nil {
		w.WriteHeader(http.StatusAccepted)

		s.inv.ResponseCh <- invocation.Response{
			StatusCode: http.StatusBadGateway,
			Header:     nil,
			Body:       body,
			Error:      errors.New(string(body)),
		}

		log.Printf("[%s] invocation [%s] failed after %s", s.id, s.inv.ID, time.Since(s.lastNext))
	} else {
		w.WriteHeader(http.StatusInternalServerError)

		resp := invocation.ResponseMessage(http.StatusInternalServerError, "could not read lambda invocation error response")
		resp.Error = err

		s.inv.ResponseCh <- resp

		log.Printf("[%s] could not read invocation [%s] error response: %+v", s.id, s.inv.ID, err)
	}

	close(s.inv.ResponseCh)
	s.doneCh <- struct{}{}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.state = idle
	s.inv = nil
}

func (s *Server) serveInvocationResponse(w http.ResponseWriter, r *http.Request) {
	if body, err := io.ReadAll(r.Body); err == nil {
		w.WriteHeader(http.StatusAccepted)

		s.inv.ResponseCh <- invocation.Response{
			StatusCode: http.StatusOK,
			Header:     nil,
			Body:       body,
			Error:      nil,
		}

		log.Printf("[%s] invocation [%s] completed in %s", s.id, s.inv.ID, time.Since(s.lastNext))
	} else {
		w.WriteHeader(http.StatusInternalServerError)

		resp := invocation.ResponseJSON(http.StatusInternalServerError, "cannot read lambda invocation response")
		resp.Error = err

		s.inv.ResponseCh <- resp

		log.Printf("[%s] could not read invocation [%s] response: %+v", s.id, s.inv.ID, err)
	}

	close(s.inv.ResponseCh)
	s.doneCh <- struct{}{}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.state = idle
	s.inv = nil
}
