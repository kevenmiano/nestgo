package runtime

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/metadata"
)

type namedSecurityError string

func (e namedSecurityError) Error() string { return string(e) }

type structSecurityError struct{}

func (structSecurityError) Error() string { return "PRIVATE_STRUCT_SENTINEL" }

type pointerSecurityError struct{}

func (*pointerSecurityError) Error() string { return "PRIVATE_POINTER_SENTINEL" }

type securityRegressionController struct{}

func (*securityRegressionController) Named() (string, namedSecurityError) {
	return "", namedSecurityError("PRIVATE_NAMED_SENTINEL")
}
func (*securityRegressionController) Struct() (string, structSecurityError) {
	return "", structSecurityError{}
}
func (*securityRegressionController) NilPointer() (string, *pointerSecurityError) {
	return "safe", nil
}
func (*securityRegressionController) Regular() (string, error) {
	return "", errors.New("PRIVATE_REGULAR_SENTINEL")
}

func securityRegressionPackages() []metadata.Package {
	return []metadata.Package{{Controllers: []metadata.Controller{{
		Type:   reflect.TypeOf((*securityRegressionController)(nil)),
		Prefix: "/security",
		Routes: []metadata.Route{
			{Method: http.MethodGet, Path: "/named", Handler: "Named"},
			{Method: http.MethodGet, Path: "/struct", Handler: "Struct"},
			{Method: http.MethodGet, Path: "/nil", Handler: "NilPointer"},
			{Method: http.MethodGet, Path: "/regular", Handler: "Regular"},
			{Method: http.MethodGet, Path: "/static", Handler: "NilPointer", StaticInvoke: func(any) (any, error) {
				return nil, errors.New("PRIVATE_STATIC_SENTINEL")
			}},
			{Method: http.MethodGet, Path: "/static-http", Handler: "NilPointer", StaticHTTP: func(any, httpadapter.Context) (any, int, error) {
				return nil, 503, errors.New("PRIVATE_HTTP_SENTINEL")
			}},
		},
	}}}}
}

func TestSecurityAdapterErrorRedactionAndNilableErrorTypes(t *testing.T) {
	a := &capturedAdapter{}
	if err := registerRoutes(a, securityRegressionPackages()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path   string
		status int
		body   string
	}{
		{"/named", 500, "Internal server error\n"},
		{"/struct", 500, "Internal server error\n"},
		{"/nil", 200, "\"safe\"\n"},
		{"/regular", 500, "Internal server error\n"},
		{"/static", 500, "Internal server error\n"},
		{"/static-http", 503, "Internal server error\n"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			ctx := &capturedContext{}
			if err := a.routes[http.MethodGet+" /security"+tc.path](ctx); err != nil {
				t.Fatal(err)
			}
			if ctx.status != tc.status || ctx.body != tc.body {
				t.Fatalf("status=%d body=%q, want status=%d body=%q", ctx.status, ctx.body, tc.status, tc.body)
			}
			if strings.Contains(ctx.body, "PRIVATE_") {
				t.Fatal("handler error leaked in HTTP response")
			}
		})
	}
}

func TestSecurityLegacyHTTPErrorRedaction(t *testing.T) {
	h, err := New(securityRegressionPackages()...)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/named", "/struct", "/regular"} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/security"+path, nil))
			if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), "PRIVATE_") {
				t.Fatalf("unexpected HTTP response status=%d body=%q", response.Code, response.Body.String())
			}
		})
	}
}
