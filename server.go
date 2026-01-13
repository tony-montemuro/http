package main

import (
	"fmt"
	"net"
	"os"

	"github.com/tony-montemuro/http/message"
)

func handle(c net.Conn) {
	parser := message.RequestParser{Connection: c}
	_, err := parser.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not parse request: %s", err.Error())
		c.Close()
	}

	c.Close()
}

func main() {
	ln, err := net.Listen("tcp", ":8080")

	if err != nil {
		fmt.Fprintf(os.Stderr, "problem starting server: %s", err.Error())
		return
	}

	fmt.Println("Listening for connections on port 8080...")
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not accept connection: %s", err.Error())
		}
		go handle(conn)
	}
}
