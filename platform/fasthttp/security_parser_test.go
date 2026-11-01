package fasthttp

import (
	"bufio"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kevenmiano/nestgo/httpadapter"
)

// Malformed request lines must not be normalized into a valid application
// request, since a front proxy may parse them differently.
func TestSecurityMalformedRequestLineLoopback(t *testing.T) {
	adapter := New()
	var invoked atomic.Int64
	if err := adapter.Register(httpadapter.Route{
		Method: http.MethodGet,
		Path:   "/hello",
		Handler: func(ctx httpadapter.Context) error {
			invoked.Add(1)
			return ctx.Write([]byte("ok"))
		},
	}); err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- adapter.Serve(listener) }()
	defer func() {
		_ = listener.Close()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("server did not stop")
		}
	}()

	for _, requestLine := range []string{
		"GET  /hello HTTP/1.1",
		"GET /hello  HTTP/1.1",
		"GET   /hello   HTTP/1.1",
	} {
		t.Run(requestLine, func(t *testing.T) {
			conn, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
				t.Fatal(err)
			}
			if _, err := conn.Write([]byte(requestLine + "\r\nHost: localhost\r\nConnection: close\r\n\r\n")); err != nil {
				t.Fatal(err)
			}
			statusLine, err := bufio.NewReader(conn).ReadString('\n')
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(statusLine, " 400 ") {
				t.Fatalf("unexpected response to malformed request line: %q", statusLine)
			}
		})
	}
	if got := invoked.Load(); got != 0 {
		t.Fatalf("malformed HTTP requests reached the application %d times", got)
	}
}
