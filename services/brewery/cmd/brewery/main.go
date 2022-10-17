package main

import (
	"log"
	"os"

	"brewery/internal/pkg/server"
)

func run() error {
	log.Println("run() Server running ...")

	port := os.Getenv("PORT")

	svr := &server.Server{APIUrl: "https://api.openbrewerydb.org"}

	shutdownSig := svr.Start(port)

	<-shutdownSig

	return nil
}

func main() {

	if err := run(); err != nil {
		log.Printf("server run() error = %s", err)
	}
}
