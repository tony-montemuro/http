package main

import (
	"log/slog"
	"os"

	"github.com/tony-montemuro/http"
)

func handler(r http.Request, w *http.ResponseWriter) {
	w.AddServerHeader([]byte("go"))
	w.SetContentTypeHeader([]byte("text"), []byte("html"))
	w.SetBody([]byte("<!DOCTYPE html><html><head><title>Website</title></head><body><h1>Tony's Web Server</h1></body></html>"))
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	}))
	srv := http.Server{ErrorLog: logger, Handler: http.HandlerFunc(handler)}
	srv.Serve()
}
