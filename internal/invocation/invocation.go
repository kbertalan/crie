package invocation

import (
	"io"
	"net/http"

	"github.com/google/uuid"
)

type Request struct {
	Body   []byte      `json:"body"`
	Header http.Header `json:"header"`
}

type Response struct {
	StatusCode int         `json:"statusCode"`
	Header     http.Header `json:"header"`
	Body       []byte      `json:"body"`
	Error      error       `json:"-"`
}

type Invocation struct {
	ID      uuid.UUID
	Request `json:"request"`

	ResponseCh chan Response `json:"-"`
}

func FromHTTPRequest(r *http.Request) (Invocation, error) {
	var invocation Invocation

	id, err := uuid.NewRandom()
	if err != nil {
		return invocation, err
	}

	invocation.ID = id

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return invocation, err
	}
	defer r.Body.Close()

	invocation.Request.Body = body
	invocation.Request.Header = r.Header
	invocation.ResponseCh = make(chan Response, 1)

	return invocation, nil
}

func (i *Invocation) Close() error {
	if i == nil {
		return nil
	}

	close(i.ResponseCh)

	return nil
}
