package http

import (
	"errors"
	"strings"
	"testing"
)

type fakeTransport struct {
	status  int
	headers map[string][]string
	body    string
}

func (f *fakeTransport) Status(status int) { f.status = status }
func (f *fakeTransport) SetHeader(k, v string) {
	if f.headers == nil {
		f.headers = map[string][]string{}
	}
	f.headers[k] = []string{v}
}
func (f *fakeTransport) AddHeader(k, v string) {
	if f.headers == nil {
		f.headers = map[string][]string{}
	}
	f.headers[k] = append(f.headers[k], v)
}
func (f *fakeTransport) Write(p []byte) error { f.body += string(p); return nil }

func TestResponseHelpers(t *testing.T) {
	for _, tc := range []struct {
		name   string
		r      Response
		status int
		body   string
	}{
		{"json", Created(map[string]string{"name": "Ada"}).WithHeader("X-Test", "yes"), 201, "{\"name\":\"Ada\"}\n"},
		{"text", Text("hello"), 200, "hello"},
		{"no-content", NoContent(), 204, ""},
		{"redirect", Redirect("/next"), 302, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeTransport{}
			if err := Write(f, tc.r, 200); err != nil {
				t.Fatal(err)
			}
			if f.status != tc.status || f.body != tc.body {
				t.Fatalf("got %d %q", f.status, f.body)
			}
			if tc.name == "redirect" && f.headers["Location"][0] != "/next" {
				t.Fatal("missing Location")
			}
		})
	}
}
func TestHTTPErrorAndExplicitWriter(t *testing.T) {
	f := &fakeTransport{}
	err := WriteError(f, fmtWrapped())
	if err != nil || f.status != 404 || !strings.Contains(f.body, "\"statusCode\":404") {
		t.Fatalf("%v: %+v", err, f)
	}
	f = &fakeTransport{}
	w := NewWriter(f)
	w.Status(202).Header("X-Test", "yes")
	if err := w.Cookie(Cookie{Name: "sid", Value: "abc", HTTPOnly: true}); err != nil {
		t.Fatal(err)
	}
	if err := w.JSON("ok"); err != nil {
		t.Fatal(err)
	}
	if !w.Written() || f.status != 202 || len(f.headers["Set-Cookie"]) != 1 {
		t.Fatalf("%+v", f)
	}
	if err := w.Send("again"); err == nil {
		t.Fatal("double write succeeded")
	}
}
func fmtWrapped() error { return errors.Join(errors.New("detail"), NotFound("missing")) }

type singleValueTransport struct {
	f fakeTransport
}

func (c *singleValueTransport) Status(s int) {
	c.f.Status(s)
}
func (c *singleValueTransport) SetHeader(k, v string) {
	c.f.SetHeader(k, v)
}
func (c *singleValueTransport) Write(data []byte) error {
	return c.f.Write(data)
}

func TestCookiesRequireMultiHeaderSupport(t *testing.T) {
	ctx := &singleValueTransport{}
	result := OK("ok").WithCookie(Cookie{Name: "a", Value: "1"}).WithCookie(Cookie{Name: "b", Value: "2"})
	if err := Write(ctx, result, 200); err == nil {
		t.Fatal("expected multi-cookie adapter validation error")
	}
	writer := NewWriter(ctx)
	if err := writer.Cookie(Cookie{Name: "a", Value: "1"}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Cookie(Cookie{Name: "b", Value: "2"}); err == nil {
		t.Fatal("second cookie silently overwrote the first")
	}
	if err := writer.Cookie(Cookie{}); err == nil {
		t.Fatal("empty cookie name accepted")
	}
}
