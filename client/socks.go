package main

import (
	"io"
	"log"
	"net"

	"golang.org/x/xerrors"

	"github.com/Squirrel-Qiu/proxy-forward/conf"
)

type ResponseType = uint8

const (
	Success ResponseType = iota
	ServerFailed
	ConnNotAllowed
	NetworkUnreachable
	HostUnreachable
	ConnRefused
	TTLExpired
	CmdNotSupport
	AddrTypeNotSupport
)

const Version = 5

type SocksServer struct {
	net.Conn
	Auth Authentication
}

func NewSocks(listener net.Conn, auth Authentication) (net.Conn, error) {
	socksServer := &SocksServer{
		listener,
		auth,
	}

	err := socksServer.init()
	if err != nil {
		_ = socksServer.Close()
		return nil, xerrors.Errorf("new socksServer failed: %w", err)
	}

	// 3.
	dstAddr, err := socksServer.ReadDstAddr()
	if err != nil {
		return nil, xerrors.Errorf("read dst address from A failed: %w", err)
	}

	conn, err := toGate()
	if err != nil {
		return nil, xerrors.Errorf("dial to gateway failed: %w", err)
	}
	log.Println("dial to gateway ok")

	// 7.
	if _, err = conn.Write(dstAddr); err != nil {
		return nil, xerrors.Errorf("write dst address to gate failed: %w", err)
	}

	// 8. receive reply  [VER | REP |  RSV  | TYPE | BND.ADDR | BND.PORT]
	reply, err := handleReply(conn)
	if err != nil {
		return nil, xerrors.Errorf("receive reply from gateway failed: %w", err)
	}

	// 9. to A
	_, err = listener.Write(reply)
	if err != nil {
		return nil, xerrors.Errorf("write reply to A failed: %w", err)
	}

	return conn, nil
}

// 1.negotiation-receive [VER | N_METHODS | METHODS]
func (server *SocksServer) init() error {
	verMsg := make([]byte, 2)

	_, err := io.ReadFull(server, verMsg)
	if err != nil {
		return xerrors.Errorf("socks read version failed: %w", err)
	}

	if verMsg[0] != Version {
		return xerrors.Errorf("socks auth version wrong: %w", VersionErr{server.RemoteAddr(), verMsg[0]})
	}

	methods := make([]byte, verMsg[1])

	_, err = io.ReadFull(server, methods)
	if err != nil {
		return xerrors.Errorf("socks read auth methods failed: %w", err)
	}

	var coincide bool
	for _, auth := range methods {
		if auth == server.Auth.Code() {
			coincide = true
			break
		}
	}
	if !coincide {
		return xerrors.Errorf("socks auth %d not coincide: %w", server.Auth.Code(), err)
	}

	// 2.
	ok, err := server.Auth.AuthFunc(server)
	if err != nil {
		return xerrors.Errorf("socks auth failed: %w", err)
	}
	if !ok {
		return xerrors.New("socks auth failed")
	}

	return nil
}

func toGate() (conn net.Conn, err error) {
	username, password, gateAddr := conf.ConfOfB1()

	// [version 1 byte | cmd 1 byte | rsv 1 byte | addr_type 1 byte]
	conn, err = net.Dial("tcp", gateAddr)
	//conn, err := net.DialTimeout("tcp", gateAddr, 5*time.Second)
	if err != nil {
		return nil, xerrors.Errorf("dial connection to gateway failed: %w", err)
	}

	log.Println("dial to gateway")

	// 4.negotiation-send [VER | NMETHODS | METHODS]
	_, err = conn.Write([]byte{Version, 1, CommonAuth})

	// 5.receive response from gateway  [VER | METHODS]
	resp := make([]byte, 2)
	_, err = io.ReadFull(conn, resp)
	if err != nil {
		return nil, xerrors.Errorf("negotiation-resp, read resp from gateway failed: %w", err)
	}

	if resp[0] != Version {
		return nil, xerrors.Errorf("negotiation-resp, auth version wrong: %w", VersionErr{conn.RemoteAddr(), resp[0]})
	}

	if resp[1] != CommonAuth {
		return nil, xerrors.Errorf("not support commonAuth certification: %w", AuthErr{conn.RemoteAddr(), resp[1]})
	}

	// 6.write username and password to gateway
	auth := Encrypt(username, password)
	_, err = conn.Write(auth)
	if err != nil {
		return nil, xerrors.Errorf("write username and password to gateway failed: %w", err)
	}

	return conn, nil
}

// [Version 1 byte | uLen 1 byte | username 1-255 byte | pLen 1 byte | pass 1-255 byte]
func Encrypt(username, password string) []byte {
	ver := []byte{Version}

	user := []byte(username)
	pass := []byte(password)
	uL := len(user)
	pL := len(pass)

	token := make([]byte, 0)

	token = append(token, byte(uL))
	token = append(token, user...)
	token = append(token, byte(pL))
	token = append(token, pass...)

	token = append(ver, token...)
	return token
}

func handleReply(conn net.Conn) ([]byte, error) {
	resp := make([]byte, 4)

	_, err := io.ReadFull(conn, resp)
	if err != nil {
		return nil, xerrors.Errorf("read version failed: %w", err)
	}

	if resp[0] != Version {
		return nil, xerrors.Errorf("reply version wrong: %w", VersionErr{conn.RemoteAddr(), resp[0]})
	}

	if resp[1] != Success {
		return nil, xerrors.Errorf("reply's response not success, is %d", resp[1])
	}

	var b []byte
	switch resp[3] {
	case TypeIPv4:
		b = make([]byte, net.IPv4len+2)
		if _, err := io.ReadFull(conn, b); err != nil {
			return nil, xerrors.Errorf("read ipv4 addr from dst failed: %w", err)
		}

	case TypeIPv6:
		b = make([]byte, net.IPv6len+2)
		if _, err := io.ReadFull(conn, b); err != nil {
			return nil, xerrors.Errorf("read ipv6 addr from dst failed: %w", err)
		}

	case TypeDomain:
		domainLen := make([]byte, 1)
		if _, err := conn.Read(domainLen); err != nil {
			return nil, xerrors.Errorf("read domain name length from dst failed: %w", err)
		}

		b = make([]byte, domainLen[0]+2)
		if _, err := io.ReadFull(conn, b); err != nil {
			return nil, xerrors.Errorf("read domain name from dst failed: %w", err)
		}
	}

	b = append(resp, b...)

	return b, nil
}
