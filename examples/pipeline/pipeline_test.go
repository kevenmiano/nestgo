package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/platform/fasthttp"
	"github.com/kevenmiano/nestgo/platform/nethttp"
	"github.com/kevenmiano/nestgo/runtime"
)

func TestRequestPipelineExample(t *testing.T) {
	t.Setenv("DEMO_API_KEY", "test-key")
	for _, engine := range []struct {
		name string
		newAdapter func() httpadapter.Adapter
	}{
		{name: "fasthttp", newAdapter: func() httpadapter.Adapter { return fasthttp.New() }},
		{name: "nethttp", newAdapter: func() httpadapter.Adapter { return nethttp.New() }},
	} {
		t.Run(engine.name, func(t *testing.T) {
			app, err := runtime.New(
				runtime.WithRegistry("pipeline"),
				runtime.WithAdapter(engine.newAdapter()),
				runtime.WithMiddleware(func(ctx httpadapter.Context, next httpadapter.Next) error {
					ctx.SetHeader("X-Example", "pipeline")
					return next()
				}),
				runtime.WithRoutes(runtime.Route{
					Method: "GET",
					Path: "/health",
					Handler: func(ctx runtime.Context) error {
						ctx.Status(200)
						return ctx.Write([]byte("ok"))
					},
				}),
			)
			if err != nil { t.Fatal(err) }
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil { t.Fatal(err) }
			done := make(chan error, 1)
			go func() { done <- app.Serve(listener) }()
			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				if err := app.Shutdown(ctx); err != nil { t.Error(err) }
				_ = listener.Close()
				select {
				case <-done:
				case <-time.After(3*time.Second):
					t.Error("shutdown timed out")
				}
			})
			client := &http.Client{Timeout: 3*time.Second}
			for _, tc := range []struct {
				name,method,path,body,key string
				status int
				contains string
			}{
				{name:"health",method:"GET",path:"/health",status:200,contains:"ok"},
				{name:"guard-denied",method:"GET",path:"/users/ada",status:403,contains:"Forbidden"},
				{name:"guard-allowed",method:"GET",path:"/users/ada",key:"test-key",status:200,contains:"\"id\":\"ada\""},
				{name:"body-trimmed",method:"POST",path:"/users/",body:`{"name":"  Ada  "}`,key:"test-key",status:201,contains:"\"name\":\"Ada\""},
				{name:"validation-filter",method:"POST",path:"/users/",body:`{"name":" "}`,key:"test-key",status:400,contains:"name is required"},
			}{
				t.Run(tc.name,func(t *testing.T) {
					req,err:=http.NewRequest(tc.method,"http://"+listener.Addr().String()+tc.path,bytes.NewBufferString(tc.body))
					if err!=nil{t.Fatal(err)}
					if tc.key!=""{req.Header.Set("X-API-Key",tc.key)}
					if tc.body!=""{req.Header.Set("Content-Type","application/json")}
					resp,err:=client.Do(req)
					if err!=nil{t.Fatal(err)}
					body,err:=io.ReadAll(resp.Body)
					resp.Body.Close()
					if err!=nil{t.Fatal(err)}
					if resp.StatusCode!=tc.status||!strings.Contains(string(body),tc.contains){
						t.Fatalf("status=%d body=%q, wanted %d and %q",resp.StatusCode,body,tc.status,tc.contains)
					}
					if resp.Header.Get("X-Example")!="pipeline"{
						t.Errorf("global middleware was not applied")
					}
					if tc.name=="validation-filter"{
						if got:=resp.Header.Get("Content-Type");!strings.HasPrefix(got,"application/json"){t.Errorf("filter content-type=%s",got)}
						var parsed map[string]any
						if err:=json.Unmarshal(body,&parsed);err!=nil{t.Fatalf("filter response is not JSON: %v",err)}
					}
				})
			}
		})
	}
}
