package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/srijan-raghavula/colearning-agent/src/bootstrap"
	"github.com/srijan-raghavula/colearning-agent/src/routes"
)

func main() {
	app := bootstrap.NewApp()

	mux := http.NewServeMux()
	routes.Register(mux, app.Routes)

	server := &http.Server{Addr: app.Config.Address, Handler: mux}

	shutdownReady, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	go func() {
		<-shutdownReady.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
			server.Close()
		}
	}()

	log.Printf("colearning-agent API listening on %s", app.Config.Address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}

	// After the listener stops, flush the durable learning snapshot so the
	// final state is never lost to an abrupt exit.
	if app.Persist != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		app.Persist(ctx)
	}
}
