package main

import (
	"log"
	"net/http"

	"github.com/srijan-raghavula/colearning-agent/src/bootstrap"
	"github.com/srijan-raghavula/colearning-agent/src/routes"
)

func main() {
	app := bootstrap.NewApp()

	mux := http.NewServeMux()
	routes.Register(mux, app.Routes)

	log.Printf("colearning-agent API listening on %s", app.Config.Address)
	if err := http.ListenAndServe(app.Config.Address, mux); err != nil {
		log.Fatal(err)
	}
}
