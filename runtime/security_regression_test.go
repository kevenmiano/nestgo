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
func (*securityRegressionController) TypedNil() (string, error) {
	var err *pointerSecurityError
	return "safe", err
}

func securityRegressionPackages() []metadata.Package {
	return []metadata.Package{{Controllers: []metadata.Controller{{
		Type:   reflect.TypeOf((*securityRegressionController)(nil)),
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
		},
	}}}}
}

func TestSecurityRejectsConcreteErrorSignatures(t *testing.T) {
	for _, method := range []string{"Named", "Struct", "NilPointer"} {
		t.Run(method, func(t *testing.T) {
			packages := []metadata.Package{{Controllers: []metadata.Controller{{
				Type:   reflect.TypeOf((*securityRegressionController)(nil)),
				Prefix: "/security",
				Routes: []metadata.Route{{Method: http.MethodGet, Path: "/invalid", Handler: method}},
			}}}}
			if err := registerRoutes(&capturedAdapter{}, packages); err == nil || !strings.Contains(err.Error(), "signature") {
				t.Fatalf("adapter runtime accepted concrete error output: %v", err)
			}
			if _, err := New(packages...); err == nil || !strings.Contains(err.Error(), "signature") {
				t.Fatalf("legacy runtime accepted concrete error output: %v", err)
			}
		})
	}
}

func TestSecurityReflectedErrorValueIsSafeForConcreteKinds(t *testing.T) {
	for _, tc := range []struct {
		value any
		want  string
	}{
		{namedSecurityError("named"), "named"},
		{structSecurityError{}, "PRIVATE_STRUCT_SENTINEL"},
		{(*pointerSecurityError)(nil), ""},
		{error((*pointerSecurityError)(nil)), ""},
	} {
		got := reflectedHandlerError(reflect.ValueOf(tc.value))
		if tc.want == "" {
			if got != nil {
				t.Fatalf("expected nil error, got %v", got)
			}
		} else if got == nil || got.Error() != tc.want {
			t.Fatalf("wrong error: %v", got)
		}
	}
}

func TestSecurityAdapterErrorRedaction(t *testing.T) {
	a := &capturedAdapter{}
	if err := registerRoutes(a, securityRegressionPackages()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path   string
		status int
	}{
		{"/regular", 500},
		{"/typed-nil", 200},
		{"/static", 500},
		{"/static-http", 503},
	} {
		t.Run(tc.path, func(t *testing.T) {
			ctx := &capturedContext{}
			if err := a.routes[http.MethodGet+" /security"+tc.path](ctx); err != nil {
				t.Fatal(err)
			}
			expected := "Internal server error\n"
			if tc.path == "/typed-nil" {
				expected = "\"safe\"\n"
			}
			if ctx.status != tc.status || ctx.body != expected {
				t.Fatalf("status=%d body=%q, want status=%d body=%q", ctx.status, ctx.body, tc.status, expected)
			}
		})
	}
}

func TestSecurityLegacyHTTPErrorRedaction(t *testing.T) {
	h, err := New(securityRegressionPackages()...)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path, body string
		status     int
	}{
		{"/regular", "Internal server error\n", 500},
		{"/typed-nil", "\"safe\"\n", 200},
	} {
		response := httptest.NewRecorder()
		h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/security"+tc.path, nil))
		if response.Code != tc.status || response.Body.String() != tc.body {
			t.Fatalf("%s: unexpected status=%d body=%q", tc.path, response.Code, response.Body.String())
		}
	}
}
