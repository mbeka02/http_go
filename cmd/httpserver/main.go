package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/mbeka02/go_http/internal/headers"
	"github.com/mbeka02/go_http/internal/request"
	"github.com/mbeka02/go_http/internal/response"
	"github.com/mbeka02/go_http/internal/server"
)

const port = 42069

func main() {
	handler := func(w *response.Writer, req *request.Request) {
		h := headers.NewHeaders()
		h.Set("Content-Type", "text/html")
		h.Set("Connection", "close")

		switch {
		case strings.HasPrefix(req.RequestLine.RequestTarget, "/httpbin/"):
			w.WriteStatusLine(response.StatusCodeOK)
			newHeaders := headers.NewHeaders()
			newHeaders.Set("Content-Type", "application/json")
			newHeaders.Set("Connection", "close")
			newHeaders.Set("Trailer", "X-Content-SHA256")
			newHeaders.Set("Trailer", "X-Content-Length")
			w.SetHeaders(newHeaders)
			w.EnableChunkedEncoding()
			resp, _ := http.Get("https://httpbin.org" + strings.TrimPrefix(req.RequestLine.RequestTarget, "/httpbin"))
			defer resp.Body.Close()
			buff := make([]byte, 1024)
			hash := sha256.New()
			var total int
			for {
				n, err := resp.Body.Read(buff)
				if n > 0 {
					total += n
					w.WriteChunkedBody(buff[:n])
					hash.Write(buff[:n])
				}
				if err == io.EOF {
					break
				}
			}
			w.WriteChunkedBodyDone()
			trailer := headers.NewHeaders()
			trailer.Set("X-Content-SHA256", fmt.Sprintf("%x", hash.Sum(nil)))
			trailer.Set("X-Content-Length", strconv.Itoa(total))
			w.WriteTrailers(trailer)
		case strings.HasPrefix(req.RequestLine.RequestTarget, "/yourproblem"):
			w.WriteStatusLine(response.StatusCodeBadRequest)
			w.SetHeaders(h)
			w.WriteBody([]byte(`
<html>
  <head>
    <title>400 Bad Request</title>
  </head>
  <body>
    <h1>Bad Request</h1>
    <p>Your request honestly kinda sucked.</p>
  </body>
</html>
      `))
		case strings.HasPrefix(req.RequestLine.RequestTarget, "/myproblem"):

			w.WriteStatusLine(response.StatusCodeInternalServerError)
			w.SetHeaders(h)
			w.WriteBody([]byte(`
<html>
  <head>
    <title>500 Internal Server Error</title>
  </head>
  <body>
    <h1>Internal Server Error</h1>
    <p>Okay, you know what? This one is on me.</p>
  </body>
</html>
      `))
		default:
			w.WriteStatusLine(response.StatusCodeOK)
			w.SetHeaders(h)
			w.WriteBody([]byte(`
<html>
  <head>
    <title>200 OK</title>
  </head>
  <body>
    <h1>Success!</h1>
    <p>Your request was an absolute banger.</p>
  </body>
</html>
      `))
		}
	}
	server, err := server.Serve(port, handler)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer server.Close()
	log.Println("Server started on port", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}
