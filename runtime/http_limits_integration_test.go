package runtime

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kevenmiano/nestgo/httpadapter"
	fastadapter "github.com/kevenmiano/nestgo/platform/fasthttp"
	"github.com/kevenmiano/nestgo/platform/nethttp"
)

func TestHTTPAdapterResourceLimitsLoopback(t *testing.T) {
	for _, tc := range []struct {
		name string
		new  func() httpadapter.Adapter
	}{
		{"nethttp", func() httpadapter.Adapter {
			return nethttp.New(nethttp.WithLimits(httpadapter.ServerLimits{
				MaxBodyBytes: 64, ReadHeaderTimeout: 150 * time.Millisecond, ReadTimeout: 150 * time.Millisecond,
			}))
		}},
		{"fasthttp", func() httpadapter.Adapter {
			return fastadapter.New(fastadapter.WithLimits(httpadapter.ServerLimits{
				MaxBodyBytes: 64, ReadTimeout: 150 * time.Millisecond,
			}))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := tc.new()
			if err := a.Register(httpadapter.Route{
				Method: "POST", Path: "/echo",
				Handler: func(ctx httpadapter.Context) error {
					return ctx.Write(ctx.Body())
				},
			}); err != nil {
				t.Fatal(err)
			}
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- a.Serve(ln) }()
			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				if err := a.Shutdown(ctx); err != nil {
					t.Error(err)
				}
				_ = ln.Close()
				select {
				case <-done:
				case <-time.After(3 * time.Second):
					t.Error("server failed to exit")
				}
			})
			base := "http://" + ln.Addr().String()
			client := &http.Client{Timeout: 2 * time.Second}
			defer client.CloseIdleConnections()
			for _, request := range []struct {
				name string
				body string
				code int
			}{
				{"under-limit", strings.Repeat("x", 64), http.StatusOK},
				{"over-limit", strings.Repeat("x", 65), http.StatusRequestEntityTooLarge},
			} {
				t.Run(request.name, func(t *testing.T) {
					resp, err := client.Post(base+"/echo", "application/octet-stream", strings.NewReader(request.body))
					if err != nil {
						t.Fatal(err)
					}
					content, err := io.ReadAll(resp.Body)
					_ = resp.Body.Close()
					if err != nil {
						t.Fatal(err)
					}
					if resp.StatusCode != request.code {
						t.Fatalf("status=%d body=%q, want=%d", resp.StatusCode, content, request.code)
					}
				})
			}

			t.Run("chunked-over-limit", func(t *testing.T) {
				req, err := http.NewRequest("POST", base+"/echo", io.NopCloser(bytes.NewBufferString(strings.Repeat("z", 65))))
				if err != nil {
					t.Fatal(err)
				}
				req.ContentLength = -1
				resp, err := client.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				_ = resp.Body.Close()
				if resp.StatusCode != http.StatusRequestEntityTooLarge {
					t.Fatalf("chunked status=%d, want 413", resp.StatusCode)
				}
			})

			t.Run("concurrent-small-requests", func(t *testing.T) {
				var wg sync.WaitGroup
				errs := make(chan error, 8)
				for i := 0; i < 8; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						resp, err := client.Post(base+"/echo", "application/octet-stream", strings.NewReader("ok"))
						if err != nil {
							errs <- err
							return
						}
						_, _ = io.Copy(io.Discard, resp.Body)
						_ = resp.Body.Close()
						if resp.StatusCode != http.StatusOK {
							errs <- fmt.Errorf("status=%d", resp.StatusCode)
						}
					}()
				}
				wg.Wait()
				close(errs)
				for err := range errs {
					t.Error(err)
				}
			})

			t.Run("slow-incomplete-header", func(t *testing.T) {
				conn, err := net.DialTimeout("tcp", ln.Addr().String(), time.Second)
				if err != nil {
					t.Fatal(err)
				}
				defer conn.Close()
				if _, err := io.WriteString(conn, "GET /echo HTTP/1.1\r\nHost: "); err != nil {
					t.Fatal(err)
				}
				if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
					t.Fatal(err)
				}
				buf := make([]byte, 1)
				_, err = conn.Read(buf)
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					t.Fatalf("slow-header client not disconnected within deadline: %v", err)
				}
			})
		})
	}
}
