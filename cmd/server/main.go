package main

import (
	"log/slog"
	"os"

	"github.com/tony-montemuro/http"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	}))
	srv := http.Server{Port: 8080, ErrorLog: logger}
	srv.Serve()
}
