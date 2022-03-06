package main

import (
	"encoding/json"
	"net/http"
)

type TraceHTTP struct{}

type TraceResponse struct {
	Status  int    `json:"status,omitempty"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

func (h *TraceHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		resp TraceResponse
	)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, X-Auth-Token")

	defer func() {
		w.WriteHeader(resp.Status)
		b, _ := json.Marshal(resp)
		_, _ = w.Write(b)
	}()

	resp.Status = http.StatusOK
	switch {
	case r.URL.Path == "/api/v1/ping":
		resp.Message = "pong"

	default:
		resp.Status = http.StatusNotFound
		return
	}
}
