package sender

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type messagePayload struct {
	Message string `json:"message"`
}

func SendMessage(w http.ResponseWriter, status int, format string, args ...any) {
	formatted := fmt.Sprintf(format, args...)
	SendJSON(w, status, messagePayload{
		Message: formatted,
	})
}

func SendJSON(w http.ResponseWriter, status int, value any) {
	payload, err := json.Marshal(value)
	if err != nil {
		log.Printf("unable to json encode %T, err: %+v", value, err)
		payload = []byte(fmt.Sprintf(`{"message":"unable to create json for type %T"}`, value))
		status = http.StatusInternalServerError
	}

	w.Header().Add("content-type", "application/json")
	w.WriteHeader(status)
	w.Write(payload)
}
