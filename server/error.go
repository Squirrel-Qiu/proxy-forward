package main

import (
	"net"
)

type VersionErr struct {
	SourceAddr net.Addr
	SocksVersion uint8
}

type AuthMethodErr struct {
	SourceAddr net.Addr
	AuthWay    uint8
}
