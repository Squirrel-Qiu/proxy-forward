package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"

	"golang.org/x/xerrors"

	"proxy-forward/serverConf"
)

func main() {
	srcAddr := flag.String("srcAddr", "0.0.0.0", "src addr")
	srcPort := flag.Int("srcPort", 2233, "listen port")

	flag.Parse()

	listenAddr := fmt.Sprintf("%s:%d", *srcAddr, *srcPort)
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

		go handleConn(listener)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	r := bufio.NewReader(conn)

	err := handShake(r, conn)
	if err != nil {
		log.Fatalf("%+v", xerrors.Errorf("shake hand failed: %w", err))
	}

	addr, err := readAddr(r)
	if err != nil {
		log.Fatalf("%+v", xerrors.Errorf("read dst address failed: %w", err))
	}
	log.Println("start connect to dst ", addr)

	remote, err := net.Dial("tcp", addr)
	if err != nil {
		// conn2' may have 'nil' or other unexpected value as its corresponding error variable may be not 'nil'
		log.Printf("%+v", xerrors.Errorf("dial to remote failed: %w", err))
		remote.Close()
		return
	}

	//wg := new(sync.WaitGroup)
	//wg.Add(2)

	go forward(remote, conn)
	go forward(conn, remote)
}

func handShake(r *bufio.Reader, conn net.Conn) error {
	userName, password := serverConf.ConfOfB2()

	verifyBuff := make([]byte, 6)
	_, _ = r.Read(verifyBuff)

	if string(verifyBuff) != "Verify" {
		return errors.New("not verification information")
	}

	uLen, _ := r.ReadByte()
	log.Printf("userName length: %d", uLen)
	userBuff := make([]byte, uLen)
	_, _ = r.Read(userBuff)

	pLen, _ := r.ReadByte()
	log.Printf("password length: %d", pLen)
	passBuff := make([]byte, pLen)
	_, _ = r.Read(passBuff)

	if string(userBuff) == userName && string(passBuff) == password {
		_, _ = conn.Write([]byte{0})
	} else {
		_, _ = conn.Write([]byte{1})
		return xerrors.New("connection verification failed")
	}
	return nil
}

func readAddr(r *bufio.Reader) (addr string, err error) {
	connBuff := make([]byte, 4)
	_, _ = r.Read(connBuff)

	if string(connBuff) != "CONN" {
		return addr, errors.New("not connection information")
	}

	typeAddr, _ := r.ReadByte()
	log.Printf("connection type (0-domain 1-ipv4 2-ipv6): %d", typeAddr)

	// 0-domain  1-ipv4  2-ipv6
	switch {
	case typeAddr == 0:
		dLen, _ := r.ReadByte()
		log.Printf("domain length: %d", dLen)
		domainBuff := make([]byte, dLen)
		_, _ = r.Read(domainBuff)

	case typeAddr == 1:
		ipv4Buff := make([]byte, 16)
		_, _ = r.Read(ipv4Buff)
		addr = net.IPv4(ipv4Buff[12], ipv4Buff[13], ipv4Buff[14], ipv4Buff[15]).String()

	case typeAddr == 2:
		ipv6Buff := make([]byte, 16)
		_, _ = r.Read(ipv6Buff)
		addr = (net.IP)(ipv6Buff).String()
	}

	var port uint16
	binary.Read(r, binary.BigEndian, &port)

	addr = fmt.Sprintf("%v:%d", addr, port)
	return addr, nil
}

func forward(conn1 net.Conn, conn2 net.Conn) {
	defer conn1.Close()
	defer conn2.Close()

	_, err := io.Copy(conn1, conn2)
	if err != nil {
		log.Printf("%+v", xerrors.Errorf("conn2 copy to conn1 failed: %w", err))
	}
}
