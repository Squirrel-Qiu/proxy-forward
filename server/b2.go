package main

import (
	"flag"
	"log"
	"net"
	"strconv"

	"golang.org/x/xerrors"
)

func main() {
	srcAddr := flag.String("srcAddr", "0.0.0.0", "src addr")
	srcPort := flag.Int("srcPort", 2233, "listener port")

	flag.Parse()

	listenAddr := net.JoinHostPort(*srcAddr, strconv.Itoa(*srcPort))
	log.Println("listen address is:", listenAddr)

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Printf("%+v", xerrors.Errorf("listen failed: %w", err))
		return
	}

	for {
		listener, err := listener.Accept()
		if err != nil {
			log.Fatalf("%+v", xerrors.Errorf("accept failed: %w", err))
		}
		log.Println(listener.LocalAddr(), " listen from ", listener.RemoteAddr())

		go handleConn(listener, WayAuth{})
	}
}
