package httputils

import (
	"fmt"
	"net/http"
)

type responseWriter struct {
	http.ResponseWriter
	StatusCode int
}

func NewResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.StatusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) WriteJSON(message []byte) {
	rw.Header().Set("Content-Type", "application/json")
	rw.Write(message)
}

func (rw *responseWriter) Error(error error, code int) {
	rw.WriteHeader(code)
	fmt.Fprintln(rw, error.Error())
}
