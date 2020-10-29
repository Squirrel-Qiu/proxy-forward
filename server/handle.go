package main

import (
	"encoding/binary"
	"io"
	"log"
	"net"

	"golang.org/x/xerrors"
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
	Target Address
}

func handleConn(listener net.Conn, auth Authentication) {
	socksServer := &SocksServer{
		listener,
		auth,
		Address{},
	}

	//
	// 4.negotiation Version
	err := socksServer.init()
	if err != nil {
		_ = socksServer.Close()
		log.Printf("%+v", xerrors.Errorf("socks init failed: %w", err))
		return
	}

	// 6.read username and password from b1
	err = socksServer.Decrypt()
	if err != nil {
		_ = socksServer.Close()
		log.Printf("%+v", xerrors.Errorf("socks decrypt failed: %w", err))
		return
	}

	// 7.
	addr, err := socksServer.ReadAddr()
	if err != nil {
		_ = socksServer.Close()
		log.Printf("%+v", xerrors.Errorf("socks read dst address failed: %w", err))
	}

	// 8.
	err = socksServer.Reply()
	if err != nil {
		_ = socksServer.Close()
		log.Printf("%+v", xerrors.Errorf("socks reply failed: %w", err))
	}

	remote, err := net.Dial("tcp", addr.String())
	if err != nil {
		// conn2' may have 'nil' or other unexpected value as its corresponding error variable may be not 'nil'
		log.Printf("%+v", xerrors.Errorf("dial to dst failed: %w", err))
		_ = remote.Close()
		return
	}

	go forward(remote, listener)
	go forward(listener, remote)
}

// 4.read  [VER | N_METHODS | METHODS]
func (server *SocksServer) init() error {
	verMsg := make([]byte, 3)

	_, err := io.ReadFull(server, verMsg)
	if err != nil {
		return xerrors.Errorf("socks read version failed: %w", err)
	}

	if verMsg[0] != Version {
		return xerrors.Errorf("socks auth version wrong: %w", VersionErr{server.RemoteAddr(), verMsg[0]})
	}

	if verMsg[1] != 1 {
		return xerrors.New("socks N_methods is not 1")
	}

	if verMsg[2] != server.Auth.Code() {
		return xerrors.Errorf("socks auth commonAuth not coincide: %w", AuthMethodErr{server.RemoteAddr(), verMsg[2]})
	}

	// 5.write VersionResp to b1
	ok, err := server.Auth.AuthFunc(server)
	if err != nil {
		return xerrors.Errorf("socks auth failed: %w", err)
	}
	if !ok {
		return xerrors.New("socks auth failed")
	}

	return nil
}

// 6.[Version 1 byte | uLen 1 byte | username 1-255 byte | pLen 1 byte | pass 1-255 byte]
func (server *SocksServer) Decrypt() error {
	verMsg := make([]byte, 2)

	_, err := io.ReadFull(server, verMsg)
	if err != nil {
		return xerrors.Errorf("decrypt read version failed: %w", err)
	}

	if verMsg[0] != Version {
		return xerrors.Errorf("decrypt auth version wrong: %w", VersionErr{server.RemoteAddr(), verMsg[0]})
	}

	username := make([]byte, verMsg[1])
	_, err = io.ReadFull(server, username)
	if err != nil {
		return xerrors.Errorf("decrypt username failed: %w", err)
	}

	pLen := make([]byte, 1)
	_, err = server.Read(pLen)
	if err != nil {
		return xerrors.Errorf("decrypt password length failed: %w", err)
	}

	password := make([]byte, pLen[0])
	_, err = io.ReadFull(server, password)
	if err != nil {
		return xerrors.Errorf("decrypt password failed: %w", err)
	}

	ok := server.Auth.DecryptAuth(username, password)
	if !ok {
		return xerrors.Errorf("decrypt username and password is not coincide: %w", AuthMethodErr{server.RemoteAddr(), CommonAuth})
	}

	return nil
}

// 8. [VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT]
func (server *SocksServer) Reply() error {
	reply := []byte{Version, Success, 0}
	if ipv4 := server.Target.IP.To4(); ipv4 != nil {
		reply = append(reply, TypeIPv4)
		reply = append(reply, ipv4...)
	} else {
		reply = append(reply, TypeIPv6)
		reply = append(reply, server.Target.IP.To16()...)
	}

	pb := make([]byte, 2)
	binary.BigEndian.PutUint16(pb, server.Target.Port)
	reply = append(reply, pb...)

	if _, err := server.Write(reply); err != nil {
		return xerrors.Errorf("socks write reply failed: %w", err)
	}

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
