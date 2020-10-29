package main

import (
	"net"

	"golang.org/x/xerrors"
)

type AuthType = uint8

const (
	NoAuth AuthType = 0
	GssAuth AuthType = 1
	CommonAuth AuthType = 2
)

type Authentication interface {
	AuthFunc(conn net.Conn) (bool, error)
	Code() uint8
}

type WayAuth struct {}

func (WayAuth) Code() uint8 {
	return NoAuth
}

func (WayAuth) AuthFunc(conn net.Conn) (bool, error) {
	_, err := conn.Write([]byte{Version, NoAuth})
	if err != nil {
		return false, xerrors.Errorf("no password auth failed: %w", err)
	}

	return true, nil
}
