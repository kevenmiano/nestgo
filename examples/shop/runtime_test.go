package shop

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kevenmiano/nestgo/platform/nethttp"
	"github.com/kevenmiano/nestgo/runtime"
)

func TestGeneratedRoutes(t *testing.T) {
	app, err := runtime.New(runtime.WithAdapter(nethttp.New()))
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan error, 1)
	go func() { done <- app.Serve(listener) }()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := app.Shutdown(ctx); err != nil {
			t.Error(err)
		}
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("server did not stop")
		}
	}()
	client := &http.Client{Timeout: 3 * time.Second}
	for _, tc := range []struct {
		method, path, body, want string
		status                   int
	}{
		{"GET", "/users/", "", "Ada", 200},
		{"GET", "/users/abc", "", "abc", 200},
		{"POST", "/users/", `{"name":"Grace"}`, "Grace", 201},
	} {
		request, err := http.NewRequest(tc.method, "http://"+listener.Addr().String()+tc.path, strings.NewReader(tc.body))
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != tc.status || !strings.Contains(string(body), tc.want) {
			t.Fatalf("%s %s: status=%d body=%s", tc.method, tc.path, response.StatusCode, body)
		}
	}
}
