package queue

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/kbertalan/crie/internal/config"
	"github.com/kbertalan/crie/internal/invocation"
)

type q chan invocation.Invocation

func (q q) insert(ctx context.Context, inv invocation.Invocation) bool {
	c := cap(q)
	if len(q) == c && c != 0 {
		return false
	}

	select {
	case <-ctx.Done():
		return false
	case <-time.After(500 * time.Millisecond):
		return false
	case q <- inv:
		return true
	}
}

func Start(ctx context.Context, cfg config.Config, ingressCh <-chan invocation.Invocation) <-chan invocation.Invocation {
	q := make(q, max(cfg.QueueSize-1, 0))

	go func() {
		defer func() {
			for inv := range q {
				inv.ResponseCh <- invocation.Response{
					StatusCode: http.StatusTooManyRequests,
					Header: http.Header{
						"content-type": []string{"application/json"},
					},
					Body:  []byte(fmt.Sprintf(`{"message": "invocation %s rejected due to server shutdown"}%s`, inv.ID, "\n")),
					Error: nil,
				}
				close(inv.ResponseCh)
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return

			case inv, ok := <-ingressCh:
				if !ok {
					return
				}

				if ok := q.insert(ctx, inv); !ok {
					inv.ResponseCh <- invocation.Response{
						StatusCode: http.StatusTooManyRequests,
						Header: http.Header{
							"content-type": []string{"application/json"},
						},
						Body:  []byte(fmt.Sprintf(`{"message": "invocation %s rejected due to too many pendding invocations"}%s`, inv.ID, "\n")),
						Error: nil,
					}
					close(inv.ResponseCh)
				}
			}
		}
	}()

	return q
}
