package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"golang.org/x/xerrors"
)

func main() {
	srcAddr := flag.String("srcAddr", "0.0.0.0", "src addr")
	srcPort := flag.Int("srcPort", 2233, "listen port")
	dstAddr := flag.String("desAddr", "1.1.1.1", "des addr")
	dstPort := flag.Int("desPort", 22, "des port")

	flag.Parse()

	listenAddr := fmt.Sprintf("%s:%d", *srcAddr, *srcPort)
	dialAddr := fmt.Sprintf("%s:%d", *dstAddr, *dstPort)

	log.Println("listen address is:", listenAddr)

	listen, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Printf("%+v", xerrors.Errorf("listen failed: %w", err))
		return
	}

	for {
		conn1, err := listen.Accept()
		if err != nil {
			log.Printf("%+v", xerrors.Errorf("accept failed: %w", err))
		}
		log.Println(conn1.LocalAddr().String(), " listen from ", conn1.RemoteAddr().String())

		go func(dialAddr string) {
			log.Println("dial connect to host:", dialAddr)

			conn2, err := net.Dial("tcp", dialAddr)
			if err != nil {
				conn1.Close()
				log.Println("close the connect at local:", conn1.LocalAddr().String(), "and remote:", conn1.RemoteAddr().String())
				return
			}

			forward(conn1, conn2)
		}(dialAddr)
	}
}

func forward(conn1 net.Conn, conn2 net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func(src net.Conn, des net.Conn) {
		_, err := io.Copy(conn2, conn1)
		if err != nil {
			log.Printf("%+v", xerrors.Errorf("conn1 copy to conn2 failed: %w", err))
		}
		conn2.Close()
		wg.Done()
	}(conn1, conn2)

	go func(src net.Conn, des net.Conn) {
		_, err := io.Copy(conn1, conn2)
		if err != nil {
			log.Printf("%+v", xerrors.Errorf("conn2 copy to conn1 failed: %w", err))
		}
		conn1.Close()
		wg.Done()
	}(conn2, conn1)

	wg.Wait()
}
