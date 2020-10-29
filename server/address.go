package main

import (
	"encoding/binary"
	"io"
	"net"
	"strconv"

	"golang.org/x/xerrors"
)

type AddressType = uint8

const (
	TypeIPv4   AddressType = 1
	TypeIPv6   AddressType = 3
	TypeDomain AddressType = 4
)

type Address struct {
	Type AddressType
	IP   net.IP
	Host string
	Port uint16
}

// 7. [Version 1 byte | cmd 1 byte | rsv 1 byte | addr_type 1 byte | dst.addr | dst.port]
func (server *SocksServer) ReadAddr() (Address, error) {
	reqMsg := make([]byte, 4)
	if _, err := io.ReadFull(server, reqMsg); err != nil {
		return Address{}, xerrors.Errorf("read dst address: read cmdConn from io.Reader failed: %w", err)
	}

	if reqMsg[0] != Version {
		return Address{}, xerrors.Errorf("socks auth version wrong: %w", VersionErr{server.RemoteAddr(), reqMsg[0]})
	}

	if reqMsg[1] != cmdConnect {
		return Address{}, xerrors.Errorf("socks only support cmdConnect, not (cmdBind=2 cmdUDPAssociate=3) %d", reqMsg[1])
	}

	var address Address
	address.Type = reqMsg[3]

	var b []byte
	switch reqMsg[3] {
	case TypeIPv4:
		b = make([]byte, net.IPv4len+2)
		if _, err := io.ReadFull(server, b); err != nil {
			return Address{}, xerrors.Errorf("read dst address ipv4 from b1 failed: %w", err)
		}

		address.IP = b[:net.IPv4len]

	case TypeIPv6:
		b = make([]byte, net.IPv6len+2)
		if _, err := io.ReadFull(server, b); err != nil {
			return Address{}, xerrors.Errorf("read dst address ipv6 from b1 failed: %w", err)
		}

		address.IP = b[:net.IPv6len]

	case TypeDomain:
		domainLen := make([]byte, 1)
		if _, err := server.Read(domainLen); err != nil {
			return Address{}, xerrors.Errorf("read dst address domain length from b1 failed: %w", err)
		}

		b = make([]byte, domainLen[0]+2)
		if _, err := io.ReadFull(server, b); err != nil {
			return Address{}, xerrors.Errorf("read dst address domain from b1 failed: %w", err)
		}

		l := int(domainLen[0])
		address.Host = string(b[:l])
		b = b[l:]

	default:
		return Address{}, xerrors.New("read invalid address type")
	}

	address.Port = binary.BigEndian.Uint16(b)

	return address, nil
}

// join host and port
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
