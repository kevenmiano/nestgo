package runtime

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	goruntime "runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/kevenmiano/nestgo/httpadapter"
	fastadapter "github.com/kevenmiano/nestgo/platform/fasthttp"
	netadapter "github.com/kevenmiano/nestgo/platform/nethttp"
)

// This helper is invoked as a real subprocess, rather than simulating
// Shutdown() from inside the same test process.
func TestOperationalSIGTERMWorker(t *testing.T) {
	engine := os.Getenv("NESTGO_OPERATIONAL_CHILD")
	if engine == "" {
		return
	}
	var adapter httpadapter.Adapter
	switch engine {
	case "fasthttp":
		adapter = fastadapter.New()
	case "nethttp":
		adapter = netadapter.New()
	default:
		t.Fatalf("unsupported worker engine %q", engine)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer stop()
	var ready atomic.Bool
	ready.Store(true)
	app, err := New(WithAdapter(adapter), WithRoutes(
		Route{Method: "GET", Path: "/health/live", Handler: func(c Context) error {
			c.Status(http.StatusOK)
			return c.Write([]byte("alive"))
		}},
		Route{Method: "GET", Path: "/health/ready", Handler: func(c Context) error {
			if !ready.Load() {
				c.Status(http.StatusServiceUnavailable)
				return c.Write([]byte("not ready"))
			}
			c.Status(http.StatusOK)
			return c.Write([]byte("ready"))
		}},
		Route{Method: "GET", Path: "/internal/fail", Handler: func(Context) error {
			return errors.New("SENSITIVE_TOKEN_SHOULD_NOT_LEAK")
		}},
		Route{Method: "GET", Path: "/inflight", Handler: func(c Context) error {
			fmt.Println("NESTGO_INFLIGHT_STARTED")
			time.Sleep(500 * time.Millisecond)
			c.Status(http.StatusOK)
			return c.Write([]byte("inflight-complete"))
		}},
	))
	if err != nil {
		t.Fatalf("initialize worker: %v", err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- app.Serve(listener) }()
	fmt.Printf("NESTGO_READY=%s\n", listener.Addr().String())
	<-ctx.Done()
	ready.Store(false)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if err := app.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintln(os.Stderr, "shutdown:", err)
		os.Exit(3)
	}
	select {
	case <-serveDone:
	case <-time.After(4 * time.Second):
		fmt.Fprintln(os.Stderr, "serve did not exit after SIGTERM")
		os.Exit(4)
	}
	fmt.Println("NESTGO_STOPPED")
	os.Exit(0)
}

func TestOperationalSIGTERMRealProcess(t *testing.T) {
	if goruntime.GOOS == "windows" {
		t.Skip("SIGTERM is not available on Windows")
	}
	for _, engine := range []string{"fasthttp", "nethttp"} {
		t.Run(engine, func(t *testing.T) {
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			command := exec.Command(executable, "-test.run=^TestOperationalSIGTERMWorker$", "-test.v")
			command.Env = append(os.Environ(), "NESTGO_OPERATIONAL_CHILD="+engine)
			stdout, err := command.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			var stderr bytes.Buffer
			command.Stderr = &stderr
			if err := command.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if command.ProcessState == nil {
					_ = command.Process.Kill()
					_ = command.Wait()
				}
			})
			addresses := make(chan string, 1)
			inflightStarted := make(chan struct{}, 1)
			stopped := make(chan bool, 1)
			go func() {
				scanner := bufio.NewScanner(stdout)
				for scanner.Scan() {
					line := scanner.Text()
					if addr, ok := strings.CutPrefix(line, "NESTGO_READY="); ok {
						addresses <- addr
					}
					if line == "NESTGO_INFLIGHT_STARTED" {
						inflightStarted <- struct{}{}
					}
					if line == "NESTGO_STOPPED" {
						stopped <- true
					}
				}
			}()
			var address string
			select {
			case address = <-addresses:
			case <-time.After(12 * time.Second):
				t.Fatalf("worker did not become ready; stderr=%s", stderr.String())
			}
			client := &http.Client{Timeout: 3 * time.Second}
			for _, tc := range []struct {
				path   string
				status int
				body   string
			}{
				{"/health/live", 200, "alive"},
				{"/health/ready", 200, "ready"},
				{"/internal/fail", 500, "Internal server error"},
			} {
				response, err := client.Get("http://" + address + tc.path)
				if err != nil {
					t.Fatalf("GET %s: %v", tc.path, err)
				}
				body, err := io.ReadAll(response.Body)
				response.Body.Close()
				if err != nil {
					t.Fatal(err)
				}
				if response.StatusCode != tc.status || !strings.Contains(string(body), tc.body) {
					t.Fatalf("%s: status=%d body=%q, want %d %q", tc.path, response.StatusCode, body, tc.status, tc.body)
				}
				if strings.Contains(string(body), "SENSITIVE_TOKEN_SHOULD_NOT_LEAK") {
					t.Fatalf("HTTP response exposed internal error details on %s", tc.path)
				}
			}
			type inflightResult struct {
				body   string
				status int
				err    error
			}
			inflightDone := make(chan inflightResult, 1)
			go func() {
				response, err := client.Get("http://" + address + "/inflight")
				if err != nil {
					inflightDone <- inflightResult{err: err}
					return
				}
				body, readErr := io.ReadAll(response.Body)
				_ = response.Body.Close()
				inflightDone <- inflightResult{body: string(body), status: response.StatusCode, err: readErr}
			}()
			select {
			case <-inflightStarted:
			case <-time.After(4 * time.Second):
				t.Fatalf("inflight handler was not entered before SIGTERM; stderr=%s", stderr.String())
			}
			if err := command.Process.Signal(syscall.SIGTERM); err != nil {
				t.Fatal(err)
			}
			select {
			case response := <-inflightDone:
				if response.err != nil || response.status != 200 || response.body != "inflight-complete" {
					t.Fatalf("inflight request interrupted by SIGTERM: status=%d body=%q err=%v", response.status, response.body, response.err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("inflight request did not finish during graceful shutdown")
			}
			client.CloseIdleConnections()
			wait := make(chan error, 1)
			go func() { wait <- command.Wait() }()
			select {
			case err := <-wait:
				if err != nil {
					t.Fatalf("worker exit: %v; stderr=%s", err, stderr.String())
				}
			case <-time.After(10 * time.Second):
				t.Fatalf("worker ignored SIGTERM; stderr=%s", stderr.String())
			}
			select {
			case <-stopped:
			case <-time.After(2 * time.Second):
				t.Fatal("worker did not report successful graceful shutdown")
			}
		})
	}
}
