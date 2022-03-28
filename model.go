package main

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
	ID    uint   `json:"id"`
	DT    string `json:"dt"`
	Level string `json:"level"`
	Type  string `json:"type"`
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}
