package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
)

var (
	ListeningAddr string
	ListeningPort int
)

func main() {
	flag.StringVar(&ListeningAddr, "b", "0.0.0.0", "Bind address")
	flag.IntVar(&ListeningPort, "p", 8080, "Port")
	flag.Parse()

	log.Println("Listening addr: " + ListeningAddr)
	log.Println("Listening port: ", ListeningPort)

	server := &Server{
		host: ListeningAddr,
		port: ListeningPort,
	}
	go server.Run()

	wait := make(chan os.Signal, 1)
	signal.Notify(wait, os.Interrupt)

	<-wait // Wait for interrupt signal
	log.Println("Shutting down...")
	server.Close()
}
