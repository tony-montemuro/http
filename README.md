# http

HTTP server implementing most of the [RFC 1945 specification](https://datatracker.ietf.org/doc/html/rfc1945), written in Go.

## Usage

**Disclaimer:** This server should not be used in a production environment! This project is primarily educational. However, it should work for basic usage.

To create an HTTP server, you will need to create an `http.Server` struct. Let's break down each `http.Server` argument:

- `Handler` (required): A struct that implements the `http.Handler` interface:
    ```go
    type Handler interface {
	    ServeHTTP(Request, *ResponseWriter)
    }
    ```

    `Request` contains information about the request, while `ResponseWriter` provides an API for setting response information. 

    To turn a normal function into a struct implementing this interface, you can use the `http.HandlerFunc` type likeso:

    ```go
    func handler(r http.Request, w *http.ResponseWriter) {
	    w.AddServerHeader([]byte("go"))
	    w.SetContentTypeHeader([]byte("text"), []byte("html"))
	    w.SetBody([]byte("<!DOCTYPE html><html><head><title>Website</title></head><body><h1>Tony's Web Server</h1></body></html>"))
    }

    h := http.HandleFunc(handler)
    ```
- `ErrorLog`: A logger of type `*slog.Logger`. See [the official Go documentation](https://pkg.go.dev/log/slog) for more information about this type. Any errors during request handling or response generation are logged using this logger.
- `MaxHeaderBytes`: A `uint16` defining the maximum nunber of bytes the server will read parsing the request headers, including the request line.
- `MaxBodyBytes`: A `uint16` defining the maximum nunber of bytes the server will read parsing the request body.
- `Port`: A `uint16` specifying the port for the server to listen on.
- `ReadTimeout`: A `uint16` specifying the amount of time the server will spend trying to read the request before timing out.

As you can see, only a `Handler` is required.

Once you have initialized a server, you just need to call the `Serve()` method to establish an HTTP server. Here is a simple example of using this server:

```go
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
```

## Testing

Before making contributions to this repository, make sure all tests pass by running the following command:

```bash
go test ./...
```

For more information on testing in Go, see [the official Go documentation](https://pkg.go.dev/testing).

## Motivations

As I said above, this project is educational in nature. As a Software Developer who primarily works on web-based products, HTTP is a technology I interact with constantly. Before this project, I had an abstract understanding of HTTP, but I didn't truly understand how it worked. Not only did this project solidify my understanding of HTTP, but also of TCP/IP, the primary network protocol that HTTP runs on-top of. Shoutout to [Beej's Guide to Network Programming](https://beej.us/guide/bgnet/html/split/index.html) for refreshing my shoddy knowledge of networking! Finally, this project taught me a lot about structuring a Go library. I actually had to refactor this project multiple times before I had an architecture I was happy with.

Overall I had a fun time working on this project. It was a bit tedious at times, but highly educational!
