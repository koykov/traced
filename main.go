package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	fcnf string
	cnf  *Config
	i10n chan os.Signal
	bus  chan []byte
)

func init() {
	var err error

	rf := func(v *string, names []string, value, usage string) {
		for i := range names {
			flag.StringVar(v, names[i], value, usage)
		}
	}
	rf(&fcnf, []string{"config", "c"}, "", "Path to config file.")
	flag.Parse()

	if len(fcnf) == 0 {
		log.Fatalln("param -config is required")
	}
	if _, err = os.Stat(fcnf); errors.Is(err, os.ErrNotExist) {
		log.Fatalf("config file '%s' doesn't exists\n", fcnf)
	}
	if cnf, err = ParseConfig(fcnf); err != nil {
		log.Fatalf("error '%s' caught on parse config '%s'\n", err.Error(), fcnf)
	}

	var la bool
	for i := 0; i < len(cnf.Listeners); i++ {
		if la = lsRepo.knowHandler(cnf.Listeners[i].Handler); la {
			break
		}
	}
	if !la {
		log.Fatalln("no listeners available")
	}

	var na bool
	for i := 0; i < len(cnf.Notifiers); i++ {
		if na = nrRepo.knowHandler(cnf.Notifiers[i].Handler); na {
			break
		}
	}
	if !na {
		log.Fatalln("no notifiers available")
	}

	if err = dbConnect(&cnf.DB); err != nil {
		log.Fatalf("couldn't connect to DB: %s\n", err.Error())
	}

	i10n = make(chan os.Signal, 1)
	signal.Notify(i10n, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
}

func main() {
	bus = make(chan []byte, cnf.BufSize)

	for i := 0; i < len(cnf.Listeners); i++ {
		l := &cnf.Listeners[i]
		if lsRepo.knowHandler(l.Handler) {
			ctx, cancel := context.WithCancel(context.Background())
			listener := lsRepo.makeListener(l)
			log.Printf("starting listener '%s' at '%s'\n", l.Handler, l.Addr)
			go func() {
				if err := listener.Listen(ctx, bus); err != nil {
					log.Printf("listener '%s' failed to start at '%s' with error '%s'\n", l.Handler, l.Addr, err.Error())
					return
				}
			}()
			lsRepo.addLC(listener, cancel)
		}
	}

	for i := 0; i < len(cnf.Notifiers); i++ {
		n := &cnf.Notifiers[i]
		if nrRepo.knowHandler(n.Handler) {
			nrRepo.makeNotifier(n)
		}
	}

	for i := uint(0); i < cnf.Workers; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		w := wsRepo.makeWorker(ctx, cancel, cnf)
		go w.work(bus)
	}

	<-i10n
	lsRepo.stopAll()
	wsRepo.stopAll()
	_ = dbClose()
	log.Println("Bye!")
}
