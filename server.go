package http

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"
)

type Handler interface {
	ServeHTTP(Request, *ResponseWriter)
}

type HandlerFunc func(Request, *ResponseWriter)

func (h HandlerFunc) ServeHTTP(r Request, w *ResponseWriter) {
	h(r, w)
}

type Server struct {
	Handler        Handler
	ErrorLog       *slog.Logger
	MaxHeaderBytes uint16
	MaxBodyBytes   uint64
	Port           uint16
	ReadTimeout    uint16
}

func (s *Server) Serve() {
	err := s.init()
	if err != nil {
		s.ErrorLog.Error(err.Error())
		return
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

func (s *Server) init() error {
	if s.ErrorLog == nil {
		s.ErrorLog = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
	if s.Handler == nil {
		return errors.New("no handler specified")
	}
	if s.Port == 0 {
		s.Port = 8080
	}
	if s.ReadTimeout == 0 {
		s.ReadTimeout = 5000
	}
	if s.MaxHeaderBytes == 0 {
		s.MaxHeaderBytes = 4000
	}
	if s.MaxBodyBytes == 0 {
		s.MaxBodyBytes = 64000
	}

	return nil
}

func (s Server) handle(c net.Conn) {
	request, err := parseRequest(c, s)
	if err != nil {
		s.ErrorLog.Error(err.Error())
		s.send(c, getErrorResponse(err))
		return
	}

	w := ResponseWriter{response: getDefaultResponse()}
	s.Handler.ServeHTTP(*request, &w)

	err = prepareBody(request, &w)
	if err != nil {
		s.ErrorLog.Error(err.Error())
		w.response = getErrorResponse(err)
	}

	s.send(c, w.response)
}

func (s Server) send(c net.Conn, r response) {
	marshaled := r.marshal()
	_, err := c.Write(marshaled)
	if err != nil {
		s.ErrorLog.Error("could not send data:", slog.String("message", err.Error()))
	}

	c.Close()
}

func prepareBody(r *Request, w *ResponseWriter) error {
	var err error
	var body []byte

	if r.Line.Method == MethodHead || w.response.code == StatusNotModified {
		body = []byte{}
	} else {
		body, err = encodeRequestBody(w.response.body, w.response.headers.contentEncoding)
	}

	w.response.body = body
	return err
}

func getDefaultResponse() response {
	return response{
		code: StatusOK,
		headers: responseHeaders{
			date:        MessageTime{date: prepareTime(time.Now())},
			contentType: ContentType{Type: "application", Subtype: "octet-stream"},
		},
	}
}

func getErrorResponse(e error) response {
	r := getDefaultResponse()

	switch err := e.(type) {
	case ClientError:
		r.code = StatusBadRequest
		r.body = []byte(err.Error())
	case ServerError:
		r.code = StatusInternalServerError
		r.body = []byte(err.Error())
	default:
		r.code = StatusInternalServerError
		r.body = []byte(err.Error())
	}

	return r
}
