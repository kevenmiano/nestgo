package runtime

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/metadata"
)

type securityNamedError string

func (e securityNamedError) Error() string { return string(e) }

type securityStructError struct{}

func (securityStructError) Error() string { return "PRIVATE_STRUCT_SENTINEL" }

type securityPointerError struct{}

func (*securityPointerError) Error() string { return "PRIVATE_POINTER_SENTINEL" }

type securityController struct{}

func (*securityController) Named() (string, securityNamedError) {
	return "", securityNamedError("PRIVATE_NAMED_SENTINEL")
}
func (*securityController) Struct() (string, securityStructError) {
	return "", securityStructError{}
}
func (*securityController) Pointer() (string, *securityPointerError) {
	return "safe", nil
}
func (*securityController) Regular() (string, error) {
	return "", errors.New("PRIVATE_REGULAR_SENTINEL")
}
func (*securityController) TypedNil() (string, error) {
	var cause *securityPointerError
	return "safe", cause
}

func securityPackages() []metadata.Package {
	return []metadata.Package{{Controllers: []metadata.Controller{{
		Type:   reflect.TypeOf((*securityController)(nil)),
		Prefix: "/security",
		Routes: []metadata.Route{
			{Method: http.MethodGet, Path: "/regular", Handler: "Regular"},
			{Method: http.MethodGet, Path: "/typed-nil", Handler: "TypedNil"},
			{Method: http.MethodGet, Path: "/static", Handler: "Regular", StaticInvoke: func(any) (any, error) {
				return nil, errors.New("PRIVATE_STATIC_SENTINEL")
			}},
			{Method: http.MethodGet, Path: "/static-http", Handler: "Regular", StaticHTTP: func(any, httpadapter.Context) (any, int, error) {
				return nil, 503, errors.New("PRIVATE_HTTP_SENTINEL")
			}},
			{Method: http.MethodGet, Path: "/static-http-bad-request", Handler: "Regular", StaticHTTP: func(any, httpadapter.Context) (any, int, error) {
				return nil, 400, errors.New("PRIVATE_400_SENTINEL")
			}},
			{Method: http.MethodGet, Path: "/static-http-invalid-status", Handler: "Regular", StaticHTTP: func(any, httpadapter.Context) (any, int, error) {
				return nil, 200, errors.New("PRIVATE_STATUS_SENTINEL")
			}},
		},
	}}}}
}

func TestSecurityRejectsConcreteErrorSignatures(t *testing.T) {
	for _, method := range []string{"Named", "Struct", "Pointer"} {
		t.Run(method, func(t *testing.T) {
			packages := []metadata.Package{{Controllers: []metadata.Controller{{
				Type:   reflect.TypeOf((*securityController)(nil)),
				Prefix: "/security",
				Routes: []metadata.Route{{Method: http.MethodGet, Path: "/invalid", Handler: method}},
			}}}}
			if _, err := New(WithPackages(packages...)); err == nil || !strings.Contains(err.Error(), "signature") {
				t.Fatalf("New accepted invalid signature: %v", err)
			}
		})
	}
}

func TestSecurityRuntimeServerErrorsDoNotLeak(t *testing.T) {
	a := &capturedAdapter{}
	app, err := New(WithPackages(securityPackages()...), WithAdapter(a))
	if err != nil {
		t.Fatal(err)
	}
	if app == nil {
		t.Fatal("nil application")
	}
	for _, tc := range []struct {
		path   string
		status int
		body   string
	}{
		{"/regular", 500, `{"statusCode":500,"message":"Internal server error"}` + "\n"},
		{"/typed-nil", 500, `{"statusCode":500,"message":"Internal server error"}` + "\n"},
		{"/static", 500, `{"statusCode":500,"message":"Internal server error"}` + "\n"},
		{"/static-http", 503, `{"statusCode":503,"message":"Internal server error"}` + "\n"},
		{"/static-http-bad-request", 400, `{"statusCode":400,"message":"Bad Request"}` + "\n"},
		{"/static-http-invalid-status", 500, `{"statusCode":500,"message":"Internal server error"}` + "\n"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			ctx := &capturedContext{}
			handler := a.routes[http.MethodGet+" /security"+tc.path]
			if handler == nil {
				t.Fatalf("no registered route for %s", tc.path)
			}
			if err := handler(ctx); err != nil {
				t.Fatal(err)
			}
			if ctx.status != tc.status || ctx.body != tc.body || strings.Contains(ctx.body, "PRIVATE_") {
				t.Fatalf("%s: status=%d body=%q, want status=%d body=%q", tc.path, ctx.status, ctx.body, tc.status, tc.body)
			}
		})
	}
}
