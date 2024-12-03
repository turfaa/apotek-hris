package httpx

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-json-experiment/json"
)

type ErrorResponse struct {
	Message string `json:"message"`
}

func Ok(w http.ResponseWriter, v any) {
	Status(w, v, http.StatusOK)
}

func Error(w http.ResponseWriter, err error, status int) {
	Status(w, ErrorResponse{Message: err.Error()}, status)
}

func Status(w http.ResponseWriter, v any, status int) {
	if v == nil {
		w.WriteHeader(status)
		return
	}

	if err := writeJSON(w, status, v); err != nil {
		log.Println("error writing json response:", err)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.MarshalWrite(w, v); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	return nil
}
