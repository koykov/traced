package main

import (
	"context"
	"sync"

	"github.com/koykov/traceID"
	"github.com/koykov/traceID/listener"
)

type listenerNewFn func(string) traceID.Listener

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
		"http": func(addr string) traceID.Listener {
			l := listener.HTTP{}
			l.SetAddr(addr)
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

func (r *listenerRepo) makeListener(handler, addr string) traceID.Listener {
	l := lnfRepo[handler](addr)
	return l
}
