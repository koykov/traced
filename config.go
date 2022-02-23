package main

import (
	"encoding/json"
	"os"

	"github.com/koykov/traceID"
)

type Config struct {
	DB string `json:"db"`
	UI string `json:"ui"`

	BufSize uint `json:"buf_size"`
	Workers uint `json:"workers"`

	Verbose bool `json:"verbose"`

	Listeners []Listener               `json:"listeners"`
	Notifiers []traceID.NotifierConfig `json:"notifiers"`
}

type Listener struct {
	Handler string `json:"handler"`
	Addr    string `json:"addr"`
}

func ParseConfig(filepath string) (*Config, error) {
	b, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	var c Config
	if err = json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if c.BufSize == 0 {
		c.BufSize = 1
	}
	if c.Workers == 0 {
		c.Workers = 1
	}
	return &c, nil
}
