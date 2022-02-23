package main

import (
	"encoding/json"
	"os"

	"github.com/koykov/traceID"
)

type Config struct {
	DB DBConfig `json:"db"`
	UI string   `json:"ui"`

	BufSize uint `json:"buf_size"`
	Workers uint `json:"workers"`

	Verbose bool `json:"verbose"`

	Listeners []traceID.ListenerConfig `json:"listeners"`
	Notifiers []traceID.NotifierConfig `json:"notifiers"`
}

type DBConfig struct {
	Driver string `json:"driver"`
	DSN    string `json:"dsn"`
	QPT    string `json:"qpt,omitempty"`
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
