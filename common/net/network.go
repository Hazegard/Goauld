package net

import (
	"net"
	"strconv"
	"time"
)

var (
	timeout = time.Second * 2
)

func CheckHostPortAvailability(host string, port int) bool {
	address := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}
