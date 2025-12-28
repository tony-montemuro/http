package main

import (
	"fmt"
	"net"
	"os"
)

func handle(c net.Conn) {
	data := make([]byte, 1024)
	n, err := c.Read(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not read data from connection: %s", err.Error())
		c.Close()
	}

	fmt.Printf("read %d bytes from connection: %s", n, string(data[:n]))
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
