package main

import "strings"

type TraceHeader struct {
	ID string `json:"id"`
	DT string `json:"dt"`
}

type TraceTree struct {
	ID       string         `json:"id"`
	Services []TraceService `json:"services"`
}

type TraceService struct {
	ID      string
	Threads uint          `json:"threads"`
	Records []TraceRecord `json:"records"`
}

type TraceRecord struct {
	ID       uint       `json:"id"`
	ThreadID uint       `json:"threadID"`
	ChildID  uint       `json:"childID,omitempty"`
	Thread   *TraceRow  `json:"thread,omitempty"`
	Rows     []TraceRow `json:"rows,omitempty"`
}

type TraceRow struct {
	ID     uint     `json:"id"`
	DT     string   `json:"dt,omitempty"`
	Level  string   `json:"level"`
	Levels []string `json:"levels,omitempty"`
	Type   string   `json:"type,omitempty"`
	Name   string   `json:"name,omitempty"`
	Value  string   `json:"value,omitempty"`
}

func applyPlaceholders(record *TraceRecord) {
	title := record.Rows[0].Value
	for i := 1; i < len(record.Rows); i++ {
		v, r := "{"+record.Rows[i].Name+"}", record.Rows[i].Value
		title = strings.ReplaceAll(title, v, r)
	}
	record.Rows[0].Value = title
}
