package main

import (
	"io"
	"net"

	"golang.org/x/xerrors"
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

// CONN: cmdConn | Type 1b | (lenDomain) | ip | port
func ReadDstAddr(r io.Reader) ([]byte, error) {
	// handle read from A

	addrType := make([]byte, 1)
	if _, err := r.Read(addrType); err != nil {
		return nil, xerrors.Errorf("read dst address: read addr type from io.Reader failed: %w", err)
	}

	var b []byte
	switch addrType[0] {
	case TypeIPv4:
		b = make([]byte, net.IPv4len+2)
		if _, err := io.ReadFull(r, b); err != nil {
			return nil, xerrors.Errorf("read dst address: read ipv4 addr from io.Reader failed: %w", err)
		}
		b = append(addrType, b...)

	case TypeIPv6:
		b = make([]byte, net.IPv6len+2)
		if _, err := io.ReadFull(r, b); err != nil {
			return nil, xerrors.Errorf("read dst address: read ipv4 addr from io.Reader failed: %w", err)
		}
		b = append(addrType, b...)

	case TypeDomain:
		domainLen := make([]byte, 1)
		if _, err := r.Read(domainLen); err != nil {
			return nil, xerrors.Errorf("read dst address: read domain length from io.Reader failed: %w", err)
		}

		b = make([]byte, domainLen[0]+2)
		if _, err := io.ReadFull(r, b); err != nil {
			return nil, xerrors.Errorf("read dst address: read domain from io.Reader failed: %w", err)
		}
		b = append(domainLen, b...)
		b = append(addrType, b...)
		b = append([]byte{cmdConn}, b...)

	default:
		return nil, xerrors.Errorf("read dst address: not support addr type %d", addrType[0])
	}

	return b, nil
}
