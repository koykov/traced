package main

import (
	"github.com/koykov/traceID"
	"github.com/koykov/traceID/notifier"
)

type notifierNewFn func(config *traceID.NotifierConfig) traceID.Notifier

type notifierRepo struct {
	buf []traceID.Notifier
}

var (
	nrfRepo = map[string]notifierNewFn{
		"slack": func(conf *traceID.NotifierConfig) traceID.Notifier {
			n := notifier.Slack{}
			n.SetConfig(conf)
			return &n
		},
		"telegram": func(conf *traceID.NotifierConfig) traceID.Notifier {
			n := notifier.Telegram{}
			n.SetConfig(conf)
			return &n
		},
	}
	nrRepo notifierRepo
)

func (r notifierRepo) knowHandler(handler string) bool {
	_, ok := nrfRepo[handler]
	return ok
}

func (r *notifierRepo) makeNotifier(conf *traceID.NotifierConfig) {
	n := nrfRepo[conf.Handler](conf)
	r.buf = append(r.buf, n)
}
