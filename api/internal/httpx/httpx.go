// Package httpx holds small helpers shared by all HTTP handlers: JSON
// encoding, a consistent error envelope, and bounded request decoding.
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// maxBodyBytes bounds request bodies; nothing in this API needs more.
const maxBodyBytes = 1 << 20 // 1 MiB

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON writes v as a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Error writes the standard error envelope:
//
//	{"error": {"code": "email_taken", "message": "..."}}
//
// Codes are stable identifiers for clients; messages are human-readable.
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, errorBody{Error: errorDetail{Code: code, Message: message}})
}

// Decode reads a JSON request body into dst, rejecting bodies over 1 MiB
// and trailing garbage after the JSON value.
func Decode(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		return err
	}
	// A second decode must hit EOF, otherwise the body held more than one
	// JSON value.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}
