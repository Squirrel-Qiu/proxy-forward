package main

import (
	"flag"
	"io"
	"log"
	"net"
	"strconv"

	"golang.org/x/xerrors"

	"proxy-forward/conf"
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
			log.Fatalf("%+v", xerrors.Errorf("accept failed: %w", err))
		}
		log.Println(listener.LocalAddr(), " listen from ", listener.RemoteAddr())

		s := "input dstAddress and dstPort"
		b := []byte{}
		listener.Write(b)

		// read from A		how read from cmd in A???
		dst, err := ReadDstAddr(listener)
		if err != nil {
			log.Fatalf("%+v", xerrors.Errorf("read dst address from A failed: %w", err))
		}

		conn, err := toGate()
		if err != nil {
			log.Fatalf("%+v", xerrors.Errorf("dial to gateway failed: %w", err))
		}
		log.Println("dial to gateway ok")

		dstAddress := net.JoinHostPort(*dstAddr, strconv.Itoa(*dstPort))
		log.Println("send dst address: ", dstAddress)
		if _, err = conn.Write(dst); err != nil {
			log.Fatalf("%+v", xerrors.Errorf("write dst address to gate failed: %w", err))
		}

		go forward(conn, listener)
		go forward(listener, conn)
	}
}

// verify: cmdVerify | len(user) | user | len(pass) | pass
func toGate() (conn net.Conn, err error) {
	userName, password, gateAddr := conf.ConfOfB1()

	log.Println("begin dial to gateway", gateAddr)

	gateVerify := []byte{cmdVerify, byte(len(userName))}
	gateVerify = append(gateVerify, []byte(userName)...)
	gateVerify = append(gateVerify, byte(len(password)))
	gateVerify = append(gateVerify, []byte(password)...)

	conn, err = net.Dial("tcp", gateAddr)
	//conn, err := net.DialTimeout("tcp", gateAddr, 5*time.Second)
	if err != nil {
		return nil, xerrors.Errorf("dial connection to gateway failed: %w", err)
	}

	log.Println("send verification info")
	if _, err = conn.Write(gateVerify); err != nil {
		return nil, xerrors.Errorf("connection write gateVerify failed: %w", err)
	}

	return conn, nil
}

func forward(conn1 net.Conn, conn2 net.Conn) {
	defer conn1.Close()
	defer conn2.Close()

	_, err := io.Copy(conn1, conn2)
	if err != nil {
		log.Printf("%+v", xerrors.Errorf("io.copy failed: %w", err))
	}
}
