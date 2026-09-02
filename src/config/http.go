package config

import "os"

type HTTP struct {
	Address string
}

func LoadHTTP() HTTP {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return HTTP{Address: ":" + port}
}
