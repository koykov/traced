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
	Threads []TraceThread `json:"threads"`
}

type TraceThread struct {
	ID      uint          `json:"id"`
	Records []TraceRecord `json:"records"`
	Threads []TraceThread `json:"threads"`
}

type TraceRecord struct {
	ID   uint       `json:"id"`
	Rows []TraceRow `json:"rows"`
}

type TraceRow struct {
	ID    uint   `json:"id"`
	DT    string `json:"dt"`
	Level string `json:"level"`
	Type  string `json:"type"`
	Name  string `json:"name,omitempty"`
	Value string `json:"value"`
}
