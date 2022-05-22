package main

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
)

type TraceHTTP struct{}

type TraceResponse struct {
	Status  int         `json:"status,omitempty"`
	Error   string      `json:"error,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
}

var (
	reTraceView  = regexp.MustCompile(`/api/v\d+/trace/(.*)`)
	reRecordView = regexp.MustCompile(`/api/v\d+/record/(\d+)`)
)

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
		resp.Payload = "pong"

	case r.URL.Path == "/api/v1/traces":
		rows, err := dbTraceList(context.Background(), "", 0)
		if err != nil {
			resp.Status = http.StatusInternalServerError
			resp.Error = err.Error()
			return
		}
		resp.Payload = rows

	case reTraceView.MatchString(r.URL.Path):
		m := reTraceView.FindStringSubmatch(r.URL.Path)
		tree, err := dbTraceTree(context.Background(), m[1])
		if err != nil {
			resp.Status = http.StatusInternalServerError
			resp.Error = err.Error()
			return
		}
		if len(tree.Services) == 0 {
			resp.Status = http.StatusNotFound
			return
		}
		resp.Payload = tree

	case reRecordView.MatchString(r.URL.Path):
		m := reRecordView.FindStringSubmatch(r.URL.Path)
		record, err := dbTraceRecord(context.Background(), m[1], false)
		if err != nil {
			resp.Status = http.StatusInternalServerError
			resp.Error = err.Error()
			return
		}
		resp.Payload = record

	default:
		resp.Status = http.StatusNotFound
		return
	}
}
