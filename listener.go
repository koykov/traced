package main

import (
	"context"
	"sync"

	"github.com/koykov/traceID"
	"github.com/koykov/traceID/listener"
)

type listenerNewFn func(config *traceID.ListenerConfig) traceID.Listener

type listenerRepo struct {
	mux sync.Mutex
	buf []listenerCancel
}

type listenerCancel struct {
	l traceID.Listener
	c context.CancelFunc
}

var (
	lnfRepo = map[string]listenerNewFn{
		"http": func(conf *traceID.ListenerConfig) traceID.Listener {
			l := listener.HTTP{}
			l.SetConfig(conf)
			return &l
		},
	}
	lsRepo listenerRepo
)

func (r *listenerRepo) addLC(l traceID.Listener, c context.CancelFunc) {
	r.mux.Lock()
	r.buf = append(r.buf, listenerCancel{l: l, c: c})
	r.mux.Unlock()
}

func (r *listenerRepo) stopAll() {
	r.mux.Lock()
	for i := 0; i < len(r.buf); i++ {
		r.buf[i].c()
	}
	r.mux.Unlock()
}

func (r *listenerRepo) knowHandler(handler string) bool {
	_, ok := lnfRepo[handler]
	return ok
}

func (r *listenerRepo) makeListener(conf *traceID.ListenerConfig) traceID.Listener {
	l := lnfRepo[conf.Handler](conf)
	return l
}
