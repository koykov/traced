package main

type MessageHeader struct {
	ID string `json:"id"`
	DT string `json:"dt"`
}

type MessageTree struct {
	ID       string           `json:"id"`
	Services []MessageService `json:"services"`
}

type MessageService struct {
	ID      string
	Threads []MessageThread `json:"threads"`
}

type MessageThread struct {
	ID      uint            `json:"id"`
	Records []MessageRecord `json:"records"`
}

type MessageRecord struct {
	ID   uint         `json:"id"`
	Rows []MessageRow `json:"rows"`
}

type MessageRow struct {
	ID    uint   `json:"id"`
	DT    string `json:"dt"`
	Level string `json:"level"`
	Type  string `json:"type"`
	Name  string `json:"name,omitempty"`
	Value string `json:"value"`
}
