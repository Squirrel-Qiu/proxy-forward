package main

import (
	"encoding/binary"
	"io"
	"net"

	"golang.org/x/xerrors"
)

type AddressType = uint8

const (
	TypeIPv4   AddressType = 1
	TypeIPv6   AddressType = 3
	TypeDomain AddressType = 4
)

// 3.[Version 1 byte | cmd 1 byte | rsv 1 byte | addr_type 1 byte | dst.addr | dst.port]
func (server *SocksServer) ReadDstAddr() ([]byte, error) {
	request := make([]byte, 4)
	if _, err := io.ReadFull(server, request); err != nil {
		return nil, xerrors.Errorf("socks read request failed: %w", err)
	}

	if request[0] != Version {
		return nil, xerrors.Errorf("socks request version wrong: %w", VersionErr{server.LocalAddr(), request[0]})
	}

	// only support cmd connect
	if request[1] != cmdConnect {
		reply := []byte{Version, CmdNotSupport, 0}
		tcpAddr := server.LocalAddr().(*net.TCPAddr)

		if len(tcpAddr.IP) == net.IPv6len {
			reply = append(reply, TypeIPv6)
		} else {
			reply = append(reply, TypeIPv4)
		}
		reply = append(reply, tcpAddr.IP...)

		port := make([]byte, 2)
		binary.BigEndian.PutUint16(port, uint16(tcpAddr.Port))
		reply = append(reply, port...)

		_, err := server.Write(reply)
		if err != nil {
			return nil, xerrors.Errorf("socks write cmd not support response failed: %w", err)
		}
		return nil, xerrors.New("socks cmd not support")
	}

	var b []byte

	switch request[3] {
	case TypeIPv4:
		b = make([]byte, net.IPv4len+2)
		if _, err := io.ReadFull(server, b); err != nil {
			return nil, xerrors.Errorf("read dst address: read ipv4 addr from io.Reader failed: %w", err)
		}

	case TypeIPv6:
		b = make([]byte, net.IPv6len+2)
		if _, err := io.ReadFull(server, b); err != nil {
			return nil, xerrors.Errorf("read dst address: read ipv4 addr from io.Reader failed: %w", err)
		}

	case TypeDomain:
		domainLen := make([]byte, 1)
		if _, err := server.Read(domainLen); err != nil {
			return nil, xerrors.Errorf("read dst address: read domain length from io.Reader failed: %w", err)
		}

		b = make([]byte, domainLen[0]+2)
		if _, err := io.ReadFull(server, b); err != nil {
			return nil, xerrors.Errorf("read dst address: read domain from io.Reader failed: %w", err)
		}
		b = append(domainLen, b...)

	default:
		return nil, xerrors.Errorf("read dst address: not support addr type %d", request[3])
	}

	b = append(request, b...)

	return b, nil
}
