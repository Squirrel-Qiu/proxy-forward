package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"golang.org/x/xerrors"

	"proxy-forward/conf"
)

func main() {
	srcAddr := flag.String("srcAddr", "127.0.0.1", "srcAddr")
	srcPort := flag.Int("srcPort", 4444, "srcPort")
	dstAddr := flag.String("dstAddr", "", "dstAddr")
	dstPort := flag.Int("dstPort", 22, "dstPort")

	flag.Parse()

	// 读A
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

		conn, err := toGate(*dstAddr, uint16(*dstPort))
		if err != nil {
			log.Fatalf("%+v", xerrors.Errorf("dial to gateway failed: %w", err))
		}
		log.Println("dial to gateway ok")

		go forward(conn, listener)
		go forward(listener, conn)
	}

	// command input to dst
}

func toGate(dstAddr string, dstPort uint16) (conn net.Conn, err error) {
	userName, password, gateAddr := conf.ConfOfB1()

	log.Println("begin dial to gateway ", gateAddr)

	conn, err = net.Dial("tcp", gateAddr)
	//conn, err := net.DialTimeout("tcp", gateAddr, 5*time.Second)
	if err != nil {
		return nil, xerrors.Errorf("dial connection failed: %w", err)
	}

	w := bufio.NewWriter(conn)

	s := "Verify"
	buff := []byte(s)
	_, err = w.Write(buff)

	uLen := len(userName)
	_ = w.WriteByte(byte(uLen))
	_, _ = w.Write([]byte(userName))

	pLen := len(password)
	_ = w.WriteByte(byte(pLen))
	_, _ = w.Write([]byte(password))

	n, err := conn.Read(buff)

	switch {
	case err != nil:
		return nil, xerrors.Errorf("connection read verification failed: %w", err)

	case n != 1:
		return nil, xerrors.Errorf("connection read verification is illegal: %w", err)

	default:
		if buff[0] == 0 {
			log.Println("connection verification pass")
		} else {
			return nil, xerrors.New("connection verification failed")
		}
	}

	log.Printf("start to dst %s:%d", dstAddr, dstPort)
	err = toDst(w, dstAddr, dstPort)
	if err != nil {
		return nil, xerrors.Errorf("connection to dst failed: %w", err)
	}

	return conn, nil
}

func toDst(conn *bufio.Writer, dstAddr string, dstPort uint16) (err error) {
	w := bufio.NewWriter(conn)

	s := "CONN"
	buff := []byte(s)
	_, err = w.Write(buff)

	addr := net.ParseIP(dstAddr)

	// 0-domain  1-ipv4  2-ipv6
	switch {
	case addr == nil:
		// 域名校验
		_ = w.WriteByte(0)

		dLen := len(dstAddr)
		_ = w.WriteByte(byte(dLen))
		_, _ = w.Write([]byte(dstAddr))

	case strings.Contains(dstAddr, "."):
		_ = w.WriteByte(1)
		_, _ = w.Write(addr)

	case strings.Contains(dstAddr, ":"):
		_ = w.WriteByte(2)
		_, _ = w.Write(addr)
	}

	// port: uint16 to byte
	port := bytes.NewBuffer([]byte{})
	binary.Write(port, binary.BigEndian, dstPort)
	_, _ = w.Write(port.Bytes())
	return nil
}

func forward(conn1 net.Conn, conn2 net.Conn) {
	defer conn1.Close()
	defer conn2.Close()

	_, err := io.Copy(conn1, conn2)
	if err != nil {
		log.Printf("%+v", xerrors.Errorf("conn2 copy to conn1 failed: %w", err))
	}
}
