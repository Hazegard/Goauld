package main

import (
	"fmt"
	"net"
)

var (
	test = "ldflag"
)

func main() {
	host := "127.0.0.1"
	port := "22"
	ssh := net.JoinHostPort(host, port)
	targetConn, err := net.Dial("tcp", ssh)
	if err != nil {
		fmt.Printf("WSSH: failed to connect to %q:%q (%s)", host, port, "aaaaa")
		return
	}

	n, err := targetConn.Write([]byte("SSH-2.0-OpenSSH_9.8\n\n\\n"))
	res := make([]byte, 1024)
	n, err = targetConn.Read(res)
	n++
}
