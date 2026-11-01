package runtime

import (
	"context"
	"errors"
	"io"
	"net"
	stdhttp "net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kevenmiano/nestgo/http"
	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/metadata"
	fastadapter "github.com/kevenmiano/nestgo/platform/fasthttp"
	"github.com/kevenmiano/nestgo/platform/nethttp"
)

type responseTestController struct{}

func (*responseTestController) Typed() http.Response[map[string]string] {
	return http.Created(map[string]string{"name": "Ada"}).WithHeader("X-Result", "typed").WithCookie(http.Cookie{Name: "one", Value: "1"})
}

type responseDTO struct {
	ID string `json:"id"`
}

func (*responseTestController) TypedDTO() http.Response[responseDTO] {
	return http.OK(responseDTO{ID: "42"})
}
func (*responseTestController) PlainDTO() responseDTO { return responseDTO{ID: "42"} }
func (*responseTestController) DynamicResult() http.Result {
	return http.Accepted(responseDTO{ID: "42"})
}

func (*responseTestController) Decorated(header string) string { return header }
func (*responseTestController) StaticRedirect() string         { return "ignored" }
func (*responseTestController) DynamicRedirect() http.Response[any] {
	return http.Redirect("/override", 307)
}
func (*responseTestController) Explicit(w *http.Writer) {
	_ = w.Status(202).Header("X-Writer", "yes").Send("accepted")
}
func (*responseTestController) Passthrough(w *http.Writer) string {
	w.Status(202).Header("X-Pass", "yes")
	return "from-handler"
}
func (*responseTestController) Cookies() http.Response[string] {
	return http.OK("cookie").WithCookie(http.Cookie{Name: "a", Value: "1"}).WithCookie(http.Cookie{Name: "b", Value: "2"})
}
func (*responseTestController) Failure() (string, error) {
	return "", http.NotFound("user missing")
}
func (*responseTestController) GenericFailure() (string, error) { return "", errors.New("secret") }
func (*responseTestController) BuiltinError(code string) (any, error) {
	switch code {
	case "405":
		return nil, http.MethodNotAllowed()
	case "421":
		return nil, http.Misdirected()
	case "429":
		return nil, http.TooManyRequests("rate limit")
	case "503":
		return nil, http.ServiceUnavailable()
	default:
		return nil, http.BadRequest("unknown code")
	}
}

func (*responseTestController) MissingWrite(w *http.Writer) {}
func responseTestPackage() metadata.Package {
	return metadata.Package{Controllers: []metadata.Controller{{
		Type:   reflect.TypeOf((*responseTestController)(nil)),
		Prefix: "/response",
		Routes: []metadata.Route{
			{Method: "GET", Path: "/typed", Handler: "Typed"},
			{Method: "GET", Path: "/typed-dto", Handler: "TypedDTO"},
			{Method: "GET", Path: "/plain-dto", Handler: "PlainDTO"},
			{Method: "GET", Path: "/result", Handler: "DynamicResult"},
			{Method: "GET", Path: "/decorated", Handler: "Decorated", Status: 201, Headers: []metadata.Header{{Name: "X-Decorated", Value: "yes"}}, Parameters: []metadata.Parameter{{Index: 0, Kind: "Headers", Name: "X-Input"}}},
			{Method: "GET", Path: "/redirect", Handler: "StaticRedirect", RedirectURL: "/dest", RedirectStatus: 301},
			{Method: "GET", Path: "/dynamic", Handler: "DynamicRedirect", RedirectURL: "/dest", RedirectStatus: 301},
			{Method: "GET", Path: "/explicit", Handler: "Explicit", Parameters: []metadata.Parameter{{Index: 0, Kind: "Res"}}},
			{Method: "GET", Path: "/passthrough", Handler: "Passthrough", Parameters: []metadata.Parameter{{Index: 0, Kind: "Res", Name: "passthrough"}}},
			{Method: "GET", Path: "/cookies", Handler: "Cookies"},
			{Method: "GET", Path: "/error", Handler: "Failure"},
			{Method: "GET", Path: "/generic-error", Handler: "GenericFailure"},
			{Method: "GET", Path: "/builtin/{code}", Handler: "BuiltinError", Parameters: []metadata.Parameter{{Index: 0, Kind: "Param", Name: "code"}}},
			{Method: "GET", Path: "/missing-write", Handler: "MissingWrite", Parameters: []metadata.Parameter{{Index: 0, Kind: "Res"}}},
		},
	}}}
}
func TestHTTPResponseEngineParity(t *testing.T) {
	for _, engine := range []struct {
		name    string
		factory func() httpadapter.Adapter
	}{
		{"fasthttp", func() httpadapter.Adapter { return fastadapter.New() }},
		{"nethttp", func() httpadapter.Adapter { return nethttp.New() }},
	} {
		t.Run(engine.name, func(t *testing.T) {
			app, err := New(WithPackages(responseTestPackage()), WithAdapter(engine.factory()))
			if err != nil {
				t.Fatal(err)
			}
			l, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- app.Serve(l) }()
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				_ = app.Shutdown(ctx)
				<-done
			}()
			checkResponseCases(t, "http://"+l.Addr().String())
		})
	}
}
func checkResponseCases(t *testing.T, base string) {
	t.Helper()
	client := &stdhttp.Client{Timeout: 3 * time.Second, CheckRedirect: func(*stdhttp.Request, []*stdhttp.Request) error { return stdhttp.ErrUseLastResponse }}
	for _, tt := range []struct {
		name, path, input                   string
		status                              int
		body, location, header, headerValue string
		cookieCount                         int
	}{
		{"typed", "/typed", "", 201, `{"name":"Ada"}` + "\n", "", "X-Result", "typed", 1},
		{"typed-dto", "/typed-dto", "", 200, `{"id":"42"}` + "\n", "", "", "", 0},
		{"plain-dto", "/plain-dto", "", 200, `{"id":"42"}` + "\n", "", "", "", 0},
		{"result", "/result", "", 202, `{"id":"42"}` + "\n", "", "", "", 0},
		{"decorated", "/decorated", "test", 201, `"test"` + "\n", "", "X-Decorated", "yes", 0},
		{"redirect", "/redirect", "", 301, "", "/dest", "", "", 0},
		{"dynamic", "/dynamic", "", 307, "", "/override", "", "", 0},
		{"explicit", "/explicit", "", 202, "accepted", "", "X-Writer", "yes", 0},
		{"passthrough", "/passthrough", "", 202, `"from-handler"` + "\n", "", "X-Pass", "yes", 0},
		{"cookies", "/cookies", "", 200, `"cookie"` + "\n", "", "", "", 2},
		{"error", "/error", "", 404, `{"statusCode":404,"message":"user missing"}` + "\n", "", "", "", 0},
		{"generic", "/generic-error", "", 500, `{"statusCode":500,"message":"Internal server error"}` + "\n", "", "", "", 0},
		{"method-not-allowed", "/builtin/405", "", 405, `{"statusCode":405,"message":"Method Not Allowed"}` + "\n", "", "", "", 0},
		{"misdirected", "/builtin/421", "", 421, `{"statusCode":421,"message":"Misdirected Request"}` + "\n", "", "", "", 0},
		{"too-many-requests", "/builtin/429", "", 429, `{"statusCode":429,"message":"rate limit"}` + "\n", "", "", "", 0},
		{"service-unavailable", "/builtin/503", "", 503, `{"statusCode":503,"message":"Service Unavailable"}` + "\n", "", "", "", 0},
		{"missing-write", "/missing-write", "", 500, `{"statusCode":500,"message":"explicit @Res() handler did not write a response"}` + "\n", "", "", "", 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req, err := stdhttp.NewRequest("GET", base+"/response"+tt.path, nil)
			if err != nil {
				t.Fatal(err)
			}
			if tt.input != "" {
				req.Header.Set("X-Input", tt.input)
			}
			res, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(res.Body)
			res.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != tt.status || string(body) != tt.body {
				t.Fatalf("status %d body %q; want %d %q", res.StatusCode, string(body), tt.status, tt.body)
			}
			if tt.location != "" && res.Header.Get("Location") != tt.location {
				t.Fatalf("Location %q", res.Header.Get("Location"))
			}
			if tt.header != "" && res.Header.Get(tt.header) != tt.headerValue {
				t.Fatalf("%s: %q", tt.header, res.Header.Get(tt.header))
			}
			if tt.cookieCount != len(res.Header.Values("Set-Cookie")) {
				t.Fatalf("cookies: %v", res.Header.Values("Set-Cookie"))
			}
			if tt.name == "generic" && strings.Contains(string(body), "secret") {
				t.Fatal("internal error leaked")
			}
		})
	}
}
func TestHTTPResponseInvalidResArgument(t *testing.T) {
	type wrong struct{}
	_ = wrong{}
	a := &capturedAdapter{}
	p := responseTestPackage()
	p.Controllers[0].Routes[0].Parameters = []metadata.Parameter{{Index: 0, Kind: "Res"}}
	if err := registerRoutes(a, []metadata.Package{p}); err == nil {
		t.Fatal("expected bootstrap signature validation")
	}
}
