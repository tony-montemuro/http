package main

import "github.com/tony-montemuro/http"

func handler(r http.Request, w *http.ResponseWriter) {
	w.AddServerHeader([]byte("go"))
	w.SetContentTypeHeader([]byte("text"), []byte("html"))
	w.SetBody([]byte("<!DOCTYPE html><html><head><title>Website</title></head><body><h1>Tony's Web Server</h1></body></html>"))
}

func main() {
	srv := http.Server{Handler: http.HandlerFunc(handler)}
	srv.Serve()
}
