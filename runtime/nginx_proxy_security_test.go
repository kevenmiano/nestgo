package runtime

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kevenmiano/nestgo/httpadapter"
	fastadapter "github.com/kevenmiano/nestgo/platform/fasthttp"
	"github.com/kevenmiano/nestgo/platform/nethttp"
)

// This test runs a real Nginx executable, but only on 127.0.0.1. It verifies
// one explicit reverse-proxy configuration, not all possible production ingress
// or cache arrangements. Ordinary go test ./... skips it unless opted in.
func TestNginxProxySecurityBoundaries(t *testing.T) {
	if os.Getenv("NESTGO_NGINX_E2E") != "1" {
		t.Skip("set NESTGO_NGINX_E2E=1 and install nginx")
	}
	nginx, err := exec.LookPath("nginx")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name       string
		newAdapter func() httpadapter.Adapter
	}{
		{"nethttp", func() httpadapter.Adapter {
			return nethttp.New(nethttp.WithLimits(httpadapter.ServerLimits{MaxBodyBytes: 64}))
		}},
		{"fasthttp", func() httpadapter.Adapter {
			return fastadapter.New(fastadapter.WithLimits(httpadapter.ServerLimits{MaxBodyBytes: 64}))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var probeHits, echoHits, cacheHits atomic.Int64
			adapter := tc.newAdapter()
			for _, route := range []httpadapter.Route{
				{
					Method: "GET", Path: "/probe",
					Handler: func(ctx httpadapter.Context) error {
						probeHits.Add(1)
						return ctx.Write([]byte("ready"))
					},
				},
				{
					Method: "POST", Path: "/echo",
					Handler: func(ctx httpadapter.Context) error {
						echoHits.Add(1)
						return ctx.Write(ctx.Body())
					},
				},
				{
					Method: "GET", Path: "/cached/{id}",
					Handler: func(ctx httpadapter.Context) error {
						cacheHits.Add(1)
						return ctx.Write([]byte(ctx.Param("id")))
					},
				},
			} {
				if err := adapter.Register(route); err != nil {
					t.Fatal(err)
				}
			}
			upstream, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			upstreamDone := make(chan error, 1)
			go func() { upstreamDone <- adapter.Serve(upstream) }()
			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				if err := adapter.Shutdown(ctx); err != nil {
					t.Error(err)
				}
				_ = upstream.Close()
				select {
				case <-upstreamDone:
				case <-time.After(3 * time.Second):
					t.Error("upstream did not exit")
				}
			})

			reservation, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			proxyAddress := reservation.Addr().String()
			_ = reservation.Close()
			workdir := t.TempDir()
			cacheDir := filepath.Join(workdir, "cache")
			proxyTempDir := filepath.Join(workdir, "proxytemp")
			for _, dir := range []string{cacheDir, proxyTempDir} {
				if err := os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
			}
			config := fmt.Sprintf(`worker_processes 1;
pid %s;
error_log %s warn;
events { worker_connections 128; }
http {
    access_log off;
    proxy_cache_path %s levels=1:2 keys_zone=nestgo_cache:1m max_size=10m inactive=1m;
    proxy_temp_path %s 1 2;
    server_tokens off;
    client_max_body_size 128;
    client_header_buffer_size 1k;
    large_client_header_buffers 2 4k;
    client_header_timeout 1s;
    client_body_timeout 1s;
    keepalive_timeout 1s;
    proxy_read_timeout 2s;
    proxy_send_timeout 2s;
    server {
        listen %s;
        location /cached/ {
            proxy_pass http://%s;
            proxy_request_buffering on;
            proxy_http_version 1.1;
            proxy_set_header Host $host;
            proxy_set_header Connection "close";
            proxy_cache nestgo_cache;
            proxy_cache_valid 200 1m;
            proxy_cache_key "$scheme$request_method$host$request_uri";
            add_header X-Cache-Status $upstream_cache_status always;
        }
        location / {
            proxy_pass http://%s;
            proxy_request_buffering on;
            proxy_http_version 1.1;
            proxy_set_header Host $host;
            proxy_set_header Connection "close";
        }
    }
}
`, filepath.Join(workdir, "nginx.pid"), filepath.Join(workdir, "error.log"), cacheDir, proxyTempDir, proxyAddress, upstream.Addr().String(), upstream.Addr().String())
			configFile := filepath.Join(workdir, "nginx.conf")
			if err := os.WriteFile(configFile, []byte(config), 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(nginx, "-p", workdir+"/", "-c", configFile, "-g", "daemon off;")
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
			})
			client := &http.Client{
				Timeout:   2 * time.Second,
				Transport: &http.Transport{DisableKeepAlives: true},
			}
			t.Cleanup(client.CloseIdleConnections)
			base := "http://" + proxyAddress
			deadline := time.Now().Add(5 * time.Second)
			for {
				response, err := client.Get(base + "/probe")
				if err == nil {
					data, readErr := io.ReadAll(response.Body)
					_ = response.Body.Close()
					if readErr == nil && response.StatusCode == 200 && string(data) == "ready" {
						break
					}
				}
				if time.Now().After(deadline) {
					t.Fatalf("nginx did not proxy to %s: %v", upstream.Addr(), err)
				}
				time.Sleep(40 * time.Millisecond)
			}

			for _, test := range []struct {
				name  string
				bytes int
				want  int
			}{
				{"bounded-body", 64, http.StatusOK},
				{"upstream-body-limit", 65, http.StatusRequestEntityTooLarge},
				{"proxy-body-limit", 129, http.StatusRequestEntityTooLarge},
			} {
				t.Run(test.name, func(t *testing.T) {
					resp, err := client.Post(base+"/echo", "application/octet-stream", strings.NewReader(strings.Repeat("x", test.bytes)))
					if err != nil {
						t.Fatal(err)
					}
					_, _ = io.Copy(io.Discard, resp.Body)
					_ = resp.Body.Close()
					if resp.StatusCode != test.want {
						t.Fatalf("HTTP status %d, want %d", resp.StatusCode, test.want)
					}
				})
			}

			t.Run("cache-key-isolation", func(t *testing.T) {
				for _, tc := range []struct {
					id        string
					wantCache string
				}{
					{"alpha", "MISS"},
					{"alpha", "HIT"},
					{"beta", "MISS"},
					{"beta", "HIT"},
				} {
					resp, err := client.Get(base + "/cached/" + tc.id)
					if err != nil {
						logData, _ := os.ReadFile(filepath.Join(workdir, "error.log"))
						t.Fatalf("cache request %s failed: %v; nginx error log:\n%s", tc.id, err, logData)
					}
					body, err := io.ReadAll(resp.Body)
					_ = resp.Body.Close()
					if err != nil {
						t.Fatal(err)
					}
					if resp.StatusCode != 200 || string(body) != tc.id || resp.Header.Get("X-Cache-Status") != tc.wantCache {
						t.Fatalf("/cached/%s: status=%d body=%q cache=%q (want %s)", tc.id, resp.StatusCode, body, resp.Header.Get("X-Cache-Status"), tc.wantCache)
					}
				}
				if got := cacheHits.Load(); got != 2 {
					t.Fatalf("cache separation: backend called %d times, want 2", got)
				}
				// The proxy canonicalizes an extra-space request line. The
				// normalized /cached/alpha must reuse alpha, not beta's entry.
				status, err := nginxRawStatus(proxyAddress, "GET  /cached/alpha HTTP/1.1\r\nHost: 127.0.0.1\r\nConnection: close\r\n\r\n")
				if err != nil {
					t.Fatal(err)
				}
				if status != 200 || cacheHits.Load() != 2 {
					t.Fatalf("normalized cached request: status=%d backendHits=%d", status, cacheHits.Load())
				}
			})

			// Raw requests exercise Nginx parsing, not net/http's normalized client.
			for _, test := range []struct {
				name       string
				request    string
				want       int
				probeDelta int64
			}{
				// Nginx accepts the extra whitespace and forwards a canonicalized
				// /probe request. Verify exactly one dispatch, not a 400.
				{"extra-request-line-space", "GET  /probe HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n", 200, 1},
				{"duplicate-content-length", "POST /echo HTTP/1.1\r\nHost: localhost\r\nContent-Length: 0\r\nContent-Length: 5\r\nConnection: close\r\n\r\n", 400, 0},
				{"transfer-encoding-content-length", "POST /echo HTTP/1.1\r\nHost: localhost\r\nTransfer-Encoding: chunked\r\nContent-Length: 5\r\nConnection: close\r\n\r\n0\r\n\r\n", 400, 0},
				{"oversized-header", "GET /probe HTTP/1.1\r\nHost: localhost\r\nX-Big: " + strings.Repeat("z", 8192) + "\r\nConnection: close\r\n\r\n", 400, 0},
			} {
				t.Run(test.name, func(t *testing.T) {
					beforeProbe, beforeEcho := probeHits.Load(), echoHits.Load()
					status, err := nginxRawStatus(proxyAddress, test.request)
					if err != nil {
						t.Fatal(err)
					}
					if status != test.want {
						t.Errorf("HTTP status %d, want %d", status, test.want)
					}
					if got := probeHits.Load() - beforeProbe; got != test.probeDelta {
						t.Fatalf("unexpected downstream probe dispatches: %d, want %d", got, test.probeDelta)
					}
					if got := echoHits.Load() - beforeEcho; got != 0 {
						t.Fatalf("malformed request reached body endpoint %d times", got)
					}
				})
			}

			t.Run("slow-incomplete-header", func(t *testing.T) {
				conn, err := net.DialTimeout("tcp", proxyAddress, time.Second)
				if err != nil {
					t.Fatal(err)
				}
				defer conn.Close()
				if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
					t.Fatal(err)
				}
				if _, err := io.WriteString(conn, "GET /probe HTTP/1.1\r\nHost: "); err != nil {
					t.Fatal(err)
				}
				_, err = bufio.NewReader(conn).ReadString('\n')
				if e, ok := err.(net.Error); ok && e.Timeout() {
					t.Fatal("Nginx did not enforce client_header_timeout")
				}
			})
		})
	}
}

func nginxRawStatus(addr, request string) (int, error) {
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		return 0, err
	}
	if _, err := io.WriteString(conn, request); err != nil {
		return 0, err
	}
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return 0, err
	}
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid proxy response line: %q", line)
	}
	return strconv.Atoi(parts[1])
}
