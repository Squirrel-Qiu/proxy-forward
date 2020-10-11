package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"flag"
	"log"
	"net"
	"strings"

	"golang.org/x/xerrors"

	"proxy-forward/conf"
)

func main() {
	dstAddr := flag.String("dstAddr", "", "dstAddr")
	dstPort := flag.Int("dstPort", 22, "dstPort")
	//dstUser := flag.String("dstUser", "root", "dstUser")
	//dstPass := flag.String("dstPass", "", "dstPass")

	flag.Parse()

	conn, err := toGate(*dstAddr, uint16(*dstPort))
	if err != nil {
		log.Fatalf("%+v", xerrors.Errorf("dial to gateway failed: %w", err))
	}
	defer conn.Close()
	log.Println("dial to gateway ok")

	// command input to dst
	conn
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
