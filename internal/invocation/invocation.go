package invocation

import (
	"encoding/json"
	"fmt"
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

const (
	XAmzInvocationType = "X-Amz-Invocation-Type"

	InvocationTypeEvent           = "Event"
	InvocationTypeRequestResponse = "RequestResponse"
	InvocationTypeDryRun          = "DryRun"
)

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

func (i Invocation) IsEvent() bool {
	return i.Request.Header.Get(XAmzInvocationType) == InvocationTypeEvent
}

func ResponseJSON(status int, value any) Response {
	buffer, err := json.Marshal(value)
	if err != nil {
		resp := ResponseMessage(http.StatusInternalServerError, "could not convert object to json: %+v", err)
		resp.Error = err

		return resp
	}

	return Response{
		StatusCode: status,
		Header: http.Header{
			"content-type": []string{"application/json"},
		},
		Body:  buffer,
		Error: nil,
	}
}

func ResponseMessage(status int, format string, args ...any) Response {
	formatted := fmt.Sprintf(format, args...)
	return Response{
		StatusCode: status,
		Header: http.Header{
			"content-type": []string{"application/json"},
		},
		Body:  fmt.Appendf(nil, `{"message": "%s"}%s`, formatted, "\n"),
		Error: nil,
	}
}
