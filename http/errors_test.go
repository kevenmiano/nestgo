package http

import (
	"errors"
	"fmt"
	stdhttp "net/http"
	"strings"
	"testing"
)

// Every helper is exercised with a default and a custom message, then
// serialized through the same transport-neutral API used by both engines.
func TestBuiltinHTTPExceptions(t *testing.T) {
	tests := []struct {
		name   string
		status int
		create func(...string) *HTTPError
	}{
		{"BadRequest", 400, BadRequest},
		{"Unauthorized", 401, Unauthorized},
		{"Forbidden", 403, Forbidden},
		{"NotFound", 404, NotFound},
		{"MethodNotAllowed", 405, MethodNotAllowed},
		{"NotAcceptable", 406, NotAcceptable},
		{"RequestTimeout", 408, RequestTimeout},
		{"Conflict", 409, Conflict},
		{"Gone", 410, Gone},
		{"PreconditionFailed", 412, PreconditionFailed},
		{"PayloadTooLarge", 413, PayloadTooLarge},
		{"UnsupportedMediaType", 415, UnsupportedMediaType},
		{"ImATeapot", 418, ImATeapot},
		{"Misdirected", 421, Misdirected},
		{"UnprocessableEntity", 422, UnprocessableEntity},
		{"TooManyRequests", 429, TooManyRequests},
		{"InternalServerError", 500, InternalServerError},
		{"NotImplemented", 501, NotImplemented},
		{"BadGateway", 502, BadGateway},
		{"ServiceUnavailable", 503, ServiceUnavailable},
		{"GatewayTimeout", 504, GatewayTimeout},
		{"HTTPVersionNotSupported", 505, HTTPVersionNotSupported},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defaultError := tc.create()
			if defaultError.StatusCode != tc.status || defaultError.Error() != stdhttp.StatusText(tc.status) {
				t.Fatalf("default error: %#v, want %d %q", defaultError, tc.status, stdhttp.StatusText(tc.status))
			}
			customError := tc.create("custom message")
			if customError.StatusCode != tc.status || customError.Error() != "custom message" {
				t.Fatalf("custom error: %#v", customError)
			}
			for _, httpErr := range []*HTTPError{defaultError, customError} {
				f := &fakeTransport{}
				if err := WriteError(f, httpErr); err != nil {
					t.Fatal(err)
				}
				want := fmt.Sprintf("{\"statusCode\":%d,\"message\":%q}\n", tc.status, httpErr.Message)
				if f.status != tc.status || f.body != want {
					t.Fatalf("got (%d,%q), want (%d,%q)", f.status, f.body, tc.status, want)
				}
				if values := f.headers["Content-Type"]; len(values) != 1 || values[0] != "application/json" {
					t.Fatalf("missing JSON content type: %#v", f.headers)
				}
			}
		})
	}
}

func TestHTTPErrorUnwrapAndPrivateFallback(t *testing.T) {
	cause := errors.New("internal database credential")
	err := &HTTPError{StatusCode: StatusBadGateway, Message: "upstream error", Cause: cause}
	if !errors.Is(err, cause) {
		t.Fatal("Cause must be unwrapped")
	}
	f := &fakeTransport{}
	if writeErr := WriteError(f, fmt.Errorf("wrapped: %w", err)); writeErr != nil {
		t.Fatal(writeErr)
	}
	if f.status != StatusBadGateway || strings.Contains(f.body, cause.Error()) {
		t.Fatalf("cause leaked or wrong status: %d %q", f.status, f.body)
	}
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"untyped", errors.New("secret")},
		{"invalid zero", NewHTTPError(0, "secret")},
		{"invalid response status", NewHTTPError(StatusOK, "secret")},
		{"invalid high", NewHTTPError(600, "secret")},
		{"typed nil", (*HTTPError)(nil)},
		{"nil", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeTransport{}
			if err := WriteError(f, tc.err); err != nil {
				t.Fatal(err)
			}
			want := "{\"statusCode\":500,\"message\":\"Internal server error\"}\n"
			if f.status != StatusInternalServerError || f.body != want {
				t.Fatalf("got %d %q, want 500 %q", f.status, f.body, want)
			}
		})
	}
}

func TestNewHTTPErrorAndMissingMessage(t *testing.T) {
	e := NewHTTPError(StatusLocked, "resource locked")
	f := &fakeTransport{}
	if err := WriteError(f, e); err != nil {
		t.Fatal(err)
	}
	if f.status != StatusLocked || !strings.Contains(f.body, "resource locked") {
		t.Fatalf("response: %d %q", f.status, f.body)
	}
	blank := NewHTTPError(StatusNotFound, "")
	f = &fakeTransport{}
	if err := WriteError(f, blank); err != nil {
		t.Fatal(err)
	}
	if f.status != StatusNotFound || !strings.Contains(f.body, "Not Found") {
		t.Fatalf("blank error: %d %q", f.status, f.body)
	}
}
