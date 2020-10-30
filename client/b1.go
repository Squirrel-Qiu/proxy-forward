package main

import (
	"flag"
	"io"
	"log"
	"net"
	"strconv"

	"golang.org/x/xerrors"

)

func main() {
	srcAddr := flag.String("srcAddr", "0.0.0.0", "srcAddr")
	srcPort := flag.Int("srcPort", 4444, "srcPort")

	flag.Parse()

	listenAddr := net.JoinHostPort(*srcAddr, strconv.Itoa(*srcPort))
	log.Println("listen address is:", listenAddr)

	listen, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Printf("%+v", xerrors.Errorf("listen failed: %w", err))
		return
	}

	for {
		listener, err := listen.Accept()
		if err != nil {
			log.Printf("%+v", xerrors.Errorf("accept failed: %w", err))
		}
		log.Println(listener.LocalAddr(), " listen from ", listener.RemoteAddr())

		// negotiation Version
		remote, err := NewSocks(listener, WayAuth{})
		if err != nil {
			_ = remote.Close()
			log.Fatalf("%+v", err)
		}

		go forward(remote, listener)
		go forward(listener, remote)
	}
}

func forward(conn1 net.Conn, conn2 net.Conn) {
	defer conn1.Close()
	defer conn2.Close()

	_, err := io.Copy(conn1, conn2)
	if err != nil {
		log.Printf("%+v", xerrors.Errorf("io.copy failed: %w", err))
	}
}
