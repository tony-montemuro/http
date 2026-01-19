package http

import (
	"fmt"
	"log/slog"
	"net"
	"os"
)

type Handler interface {
	ServeHTTP(Request, ResponseWriter)
}

type Server struct {
	Handler  Handler
	ErrorLog *slog.Logger
	Port     uint16
}

func (s *Server) Serve() {
	if s.ErrorLog == nil {
		s.ErrorLog = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
	if s.Handler == nil {
		s.ErrorLog.Error("no handler specificed")
		return
	}
	if s.Port == 0 {
		s.Port = 8080
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		s.ErrorLog.Error("problem starting server", slog.String("error", err.Error()))
		return
	}

	fmt.Printf("Listening for connections on port %d...", s.Port)
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not accept connection: %s", err.Error())
		}
		go s.handle(conn)
	}
}

func (s Server) handle(c net.Conn) {
	request, err := parseRequest(c)
	if err != nil {
		s.ErrorLog.Error(err.Error())
		c.Close()
	}

	s.Handler.ServeHTTP(*request, ResponseWriter{response: response{}})

	c.Close()
}
