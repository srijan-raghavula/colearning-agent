package main

import (
	"log"
	"net/http"

	"github.com/srijan-raghavula/colearning-agent/internal/interfaces/httpapi"
)

func main() {
	mux := http.NewServeMux()
	httpapi.RegisterRoutes(mux)

	log.Println("colearning-agent API listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
