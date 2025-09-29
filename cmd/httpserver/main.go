package main

import (
	"log"
	"os"
	"os/signal"
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

		switch req.RequestLine.RequestTarget {
		case "/yourproblem":
			w.WriteStatusLine(response.StatusCodeBadRequest)
			w.WriteHeaders(h)
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
		case "/myproblem":
			w.WriteStatusLine(response.StatusCodeInternalServerError)
			w.WriteHeaders(h)
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
			w.WriteHeaders(h)
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
