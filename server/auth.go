package main

import (
	"log"
	"net"

	"golang.org/x/xerrors"

	"proxy-forward/serverConf"
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
	DecryptAuth(u, p []byte) bool
}

type WayAuth struct {}

func (WayAuth) Code() uint8 {
	return CommonAuth
}

func (WayAuth) AuthFunc(conn net.Conn) (bool, error) {
	_, err := conn.Write([]byte{Version, CommonAuth})
	if err != nil {
		return false, xerrors.Errorf("commonAuth of username and password auth failed: %w", err)
	}

	return true, nil
}

func (WayAuth) DecryptAuth(u, p []byte) bool {
	userName, password := serverConf.ConfOfB2()
	log.Println("read the gateway configuration and start to verify visitors")

	if string(u) == userName && string(p) == password {
		log.Println("decrypt username and password pass")
		return true
	}

	return false
}
