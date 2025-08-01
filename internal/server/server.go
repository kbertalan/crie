package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/kbertalan/crie/internal/config"
	"github.com/kbertalan/crie/internal/invocation"
)

func ListenAndServe(ctx context.Context, cfg config.Config, wg *sync.WaitGroup, cancel context.CancelFunc, invocationCh chan<- invocation.Invocation) {
	defer close(invocationCh)

	handler := http.NewServeMux()
	pattern := fmt.Sprintf("POST /2015-03-31/functions/%s/invocations", cfg.LambdaName)

	handler.Handle(pattern, &invokeHandler{
		invocationCh: invocationCh,
		cfg:          cfg,
	})

	srv := http.Server{
		Addr:    string(cfg.ServerAddress),
		Handler: handler,
	}

	go func() {
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server failed with error: %+v", err)
			cancel()
		}
	}()

	defer wg.Done()
	<-ctx.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ServerShutdownTimeout)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server graceful shutdown has failed: %+v", err)
	}
}

type invokeHandler struct {
	invocationCh chan<- invocation.Invocation
	cfg          config.Config
}

func (h *invokeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	inv, err := invocation.FromHTTPRequest(r)
	if err != nil {
		log.Printf("cannot construct invocation from request: %+v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	select {
	case h.invocationCh <- inv:

	case <-time.After(h.cfg.WaitForQueueCapacity):
		close(inv.ResponseCh)
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprintf(w, `{"message":"invocation queue is full: %s"}`, inv.ID)
		return

	case <-r.Context().Done():
		close(inv.ResponseCh)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if inv.IsEvent() {
		w.WriteHeader(http.StatusAccepted)
		go h.getResponse(inv)
		return
	}

	response := h.getResponse(inv)

	for name, values := range response.Header {
		w.Header().Del(name)
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	w.WriteHeader(response.StatusCode)
	w.Write(response.Body)
}

func (h *invokeHandler) getResponse(inv invocation.Invocation) invocation.Response {
	select {
	case response, ok := <-inv.ResponseCh:
		if !ok {
			log.Printf("[%s]: reponse channel was closed unexpectedly", inv.ID)
			return invocation.Response{
				StatusCode: http.StatusInternalServerError,
			}
		}

		if err := response.Error; err != nil {
			log.Printf("[%s]: processing request failed: %+v", inv.ID, err)
		}
		return response
	case <-time.After(h.cfg.LambdaRuntimeDeadline):
		go func() {
			<-inv.ResponseCh
		}()

		resp := invocation.ResponseMessage(http.StatusBadGateway, "lambda timeout after %s", h.cfg.LambdaRuntimeDeadline)
		resp.Error = fmt.Errorf("lambda timeout after %s", h.cfg.LambdaRuntimeDeadline)
		return resp
	}
}
