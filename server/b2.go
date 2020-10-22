package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"io"
	"log"
	"net"
	"strconv"

	"golang.org/x/xerrors"

	"proxy-forward/serverConf"
)

type cmd = uint8

const (
	cmdVerify cmd = 6
	cmdConn   cmd = 7
)

type typeAddress = uint8

const (
	TypeIPv4 typeAddress = 1
	TypeIPv6 typeAddress = 2
	TypeDomain typeAddress = 3
)

type Address struct {
	Type typeAddress
	IP net.IP
	Host string
	Port uint16
}

func main() {
	srcAddr := flag.String("srcAddr", "0.0.0.0", "src addr")
	srcPort := flag.Int("srcPort", 2233, "listen port")

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

		go handleConn(listener)
	}
}

func handleConn(conn net.Conn) {
	r := bufio.NewReader(conn)

	_, err := VerifyClient(conn)
	if err != nil {
		log.Fatalf("%+v", xerrors.Errorf("shake hand failed: %w", err))
	}

	log.Println("the client is valid, verified by the gateway")

	addr, err := readAddr(conn)
	if err != nil {
		log.Fatalf("%+v", xerrors.Errorf("read dst address failed: %w", err))
	}

	dstAddr := addr.String()
	log.Println("start connect to dst ", dstAddr)

	remote, err := net.Dial("tcp", dstAddr)
	if err != nil {
		// conn2' may have 'nil' or other unexpected value as its corresponding error variable may be not 'nil'
		log.Printf("%+v", xerrors.Errorf("dial to remote failed: %w", err))
		remote.Close()
		return
	}

	go forward(remote, conn)
	go forward(conn, remote)
}

func VerifyClient(conn net.Conn) (bool, error) {
	userName, password := serverConf.ConfOfB2()
	log.Println("read the gateway configuration and start to verify visitors")

	b := make([]byte, 1)
	_, err := conn.Read(b)
	if err != nil {
		return false, xerrors.Errorf("read verification identifier from io.Reader failed: %w", err)
	}

	if b[0] != cmdVerify {
		return false, xerrors.New("it is not verification message")
	}

	// username
	if _, err := conn.Read(b); err != nil {
		return false, xerrors.Errorf("read username length from io.Reader failed: %w", err)
	}

	userBuff := make([]byte, b[0])
	if _, err := io.ReadFull(conn, userBuff); err != nil {
		return false, xerrors.Errorf("read username from io.Reader failed: %w", err)
	}

	// password
	if _, err := conn.Read(b); err != nil {
		return false, xerrors.Errorf("read password length from io.Reader failed: %w", err)
	}

	passBuff := make([]byte, b[0])
	if _, err := io.ReadFull(conn, passBuff); err != nil {
		return false, xerrors.Errorf("read password from io.Reader failed: %w", err)
	}

	if string(userBuff) == userName && string(passBuff) == password {
		_, _ = conn.Write([]byte{})
		return true, nil
	}

	return false, xerrors.New("verify client failed")
}

func readAddr(r io.Reader) (Address, error) {
	connBuff := make([]byte, 1)
	if _, err := r.Read(connBuff); err != nil {
		return Address{}, xerrors.Errorf("read dst address: read cmdConn from io.Reader failed: %w", err)
	}

	if connBuff[0] != cmdConn {
		return Address{}, xerrors.New("it is not cmdConn(dstAddr) message")
	}

	if _, err := r.Read(connBuff); err != nil {
		return Address{}, xerrors.Errorf("read dst address: read addr type from io.Reader failed: %w", err)
	}

	var address Address

	var b []byte
	switch connBuff[0] {
	case TypeIPv4:
		b = make([]byte, net.IPv4len+2)
		if _, err := io.ReadFull(r, b); err != nil {
			return Address{}, xerrors.Errorf("read dst address: read ipv4 addr from io.Reader failed: %w", err)
		}

		address.IP = b[:net.IPv4len]

	case TypeIPv6:
		b = make([]byte, net.IPv6len+2)
		if _, err := io.ReadFull(r, b); err != nil {
			return Address{}, xerrors.Errorf("read dst address: read ipv4 addr from io.Reader failed: %w", err)
		}

		address.IP = b[:net.IPv6len]

	case TypeDomain:
		domainLen := make([]byte, 1)
		if _, err := r.Read(domainLen); err != nil {
			return Address{}, xerrors.Errorf("read dst address: read domain length from io.Reader failed: %w", err)
		}

		b = make([]byte, domainLen[0]+2)
		if _, err := io.ReadFull(r, b); err != nil {
			return Address{}, xerrors.Errorf("read dst address: read domain from io.Reader failed: %w", err)
		}

		l := domainLen[0]
		address.Host = string(b[:l])
		b = b[l:]

	default:
		return Address{}, xerrors.New("read invalid address type")
	}

	address.Port = binary.BigEndian.Uint16(b)

	return address, nil
}

func forward(conn1 net.Conn, conn2 net.Conn) {
	defer conn1.Close()
	defer conn2.Close()

	_, err := io.Copy(conn1, conn2)
	if err != nil {
		log.Printf("%+v", xerrors.Errorf("conn2 copy to conn1 failed: %w", err))
	}
}

func (address Address) String() string {
	switch address.Type {
	case TypeIPv4, TypeIPv6:
		return net.JoinHostPort(address.IP.String(), strconv.Itoa(int(address.Port)))

	case TypeDomain:
		return net.JoinHostPort(address.Host, strconv.Itoa(int(address.Port)))

	default:
		return ""
	}
}
