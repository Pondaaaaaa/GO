package main

import (
	"fmt"
	"net"
)

func listenLoopback(port int) (net.Listener, error) {
	return net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
}
