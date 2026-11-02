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
	app, err := bootstrap.NewApp()
	if err != nil {
		log.Fatalf("initialize application: %v", err)
	}
	if app.Close != nil {
		defer func() {
			if err := app.Close(); err != nil {
				log.Printf("close application dependencies: %v", err)
			}
		}()
	}

	mux := http.NewServeMux()
	routes.Register(mux, app.Routes)

	server := &http.Server{
		Addr:              app.Config.Address,
		Handler:           http.MaxBytesHandler(mux, 1<<20),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	shutdownReady, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	go func() {
		<-shutdownReady.Done()
		log.Println("graceful shutdown started (SIGINT/SIGTERM received)")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
			server.Close()
		} else {
			log.Println("graceful shutdown complete")
		}
	}()

	log.Printf("colearning-agent API listening on %s", app.Config.Address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("HTTP server stopped with error: %v", err)
	}

}
