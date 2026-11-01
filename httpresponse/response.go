// Package httpresponse provides platform-independent HTTP response values and helpers.
package httpresponse

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Transport is the minimal response-writing contract implemented by every HTTP engine.
type Transport interface {
	Status(int)
	SetHeader(string, string)
	Write([]byte) error
}

// HeaderAppender is an optional adapter capability, required to emit multiple cookies.
type HeaderAppender interface{ AddHeader(string, string) }

// Cookie contains standard HTTP cookie attributes without exposing an engine type.
type Cookie struct {
	Name, Value, Path, Domain string
	MaxAge                    int
	HTTPOnly, Secure          bool
	SameSite                  http.SameSite
}

func (c Cookie) String() string {
	return (&http.Cookie{Name: c.Name, Value: c.Value, Path: c.Path, Domain: c.Domain,
		MaxAge: c.MaxAge, HttpOnly: c.HTTPOnly, Secure: c.Secure, SameSite: c.SameSite}).String()
}

// Response is a response description, not a native server response.
type Response struct {
	StatusCode  int
	Body        any
	ContentType string
	Headers     map[string]string
	RedirectURL string
	Cookies     []Cookie
}

func JSON(body any) Response { return Response{Body: body} }
func Text(body string) Response {
	return Response{Body: body, ContentType: "text/plain; charset=utf-8"}
}
func OK(body any) Response       { return JSON(body).WithStatus(http.StatusOK) }
func Created(body any) Response  { return JSON(body).WithStatus(http.StatusCreated) }
func Accepted(body any) Response { return JSON(body).WithStatus(http.StatusAccepted) }
func NoContent() Response        { return Response{StatusCode: http.StatusNoContent} }
func Redirect(url string, status ...int) Response {
	code := http.StatusFound
	if len(status) > 0 {
		code = status[0]
	}
	return Response{StatusCode: code, RedirectURL: url}
}
func (r Response) WithStatus(code int) Response { r.StatusCode = code; return r }
func (r Response) WithHeader(name, value string) Response {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	} else {
		copyHeaders := make(map[string]string, len(r.Headers)+1)
		for k, v := range r.Headers {
			copyHeaders[k] = v
		}
		r.Headers = copyHeaders
	}
	r.Headers[name] = value
	return r
}
func (r Response) WithCookie(cookie Cookie) Response {
	r.Cookies = append(append([]Cookie(nil), r.Cookies...), cookie)
	return r
}

// HTTPError is a typed error that retains its status across HTTP platforms.
type HTTPError struct {
	StatusCode int
	Message    string
	Cause      error
}

func (e *HTTPError) Error() string { return e.Message }
func (e *HTTPError) Unwrap() error { return e.Cause }
func NewHTTPError(status int, message string) *HTTPError {
	return &HTTPError{StatusCode: status, Message: message}
}
func BadRequest(message string) *HTTPError   { return NewHTTPError(http.StatusBadRequest, message) }
func Unauthorized(message string) *HTTPError { return NewHTTPError(http.StatusUnauthorized, message) }
func Forbidden(message string) *HTTPError    { return NewHTTPError(http.StatusForbidden, message) }
func NotFound(message string) *HTTPError     { return NewHTTPError(http.StatusNotFound, message) }
func Conflict(message string) *HTTPError     { return NewHTTPError(http.StatusConflict, message) }
func UnprocessableEntity(message string) *HTTPError {
	return NewHTTPError(http.StatusUnprocessableEntity, message)
}
func InternalServerError(message string) *HTTPError {
	return NewHTTPError(http.StatusInternalServerError, message)
}

func validStatus(status int) bool { return status >= 100 && status <= 599 }

func appendCookie(ctx Transport, c Cookie) error {
	if c.Name == "" {
		return errors.New("cookie name cannot be empty")
	}
	if a, ok := ctx.(HeaderAppender); ok {
		a.AddHeader("Set-Cookie", c.String())
		return nil
	}
	ctx.SetHeader("Set-Cookie", c.String())
	return nil
}

// Write serializes a response after resolving status, headers and redirect rules.
// Ordinary return values use JSON, preserving NestGo's implicit response convention.
func Write(ctx Transport, value any, defaultStatus int) error {
	var r Response
	switch v := value.(type) {
	case Response:
		r = v
	case *Response:
		if v == nil {
			r = JSON(nil)
		} else {
			r = *v
		}
	default:
		r = JSON(value)
	}
	if r.RedirectURL != "" && r.StatusCode == 0 {
		r.StatusCode = http.StatusFound
	}
	if r.StatusCode == 0 {
		r.StatusCode = defaultStatus
	}
	if r.StatusCode == 0 {
		r.StatusCode = http.StatusOK
	}
	if !validStatus(r.StatusCode) {
		return fmt.Errorf("invalid HTTP status: %d", r.StatusCode)
	}
	if r.RedirectURL != "" && (r.StatusCode < 300 || r.StatusCode > 399) {
		return fmt.Errorf("redirect status must be 3xx: %d", r.StatusCode)
	}
	noBody := r.StatusCode == http.StatusNoContent || r.StatusCode == http.StatusNotModified || (r.StatusCode >= 100 && r.StatusCode < 200)
	var payload []byte
	if !noBody && r.RedirectURL == "" {
		if r.ContentType == "text/plain; charset=utf-8" {
			payload = []byte(fmt.Sprint(r.Body))
		} else {
			encoded, err := json.Marshal(r.Body)
			if err != nil {
				return fmt.Errorf("encode HTTP response: %w", err)
			}
			payload = append(encoded, '\n')
			if r.ContentType == "" {
				r.ContentType = "application/json"
			}
		}
	}
	if r.RedirectURL != "" {
		ctx.SetHeader("Location", r.RedirectURL)
	}
	if r.ContentType != "" && !noBody && r.RedirectURL == "" {
		ctx.SetHeader("Content-Type", r.ContentType)
	}
	for key, val := range r.Headers {
		ctx.SetHeader(key, val)
	}
	for _, cookie := range r.Cookies {
		if err := appendCookie(ctx, cookie); err != nil {
			return err
		}
	}
	ctx.Status(r.StatusCode)
	return ctx.Write(payload)
}

// WriteError preserves typed HTTP statuses and produces NestJS-like JSON errors.
// Unrecognized errors are never returned to callers with an arbitrary status code.
func WriteError(ctx Transport, err error) error {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		status := httpErr.StatusCode
		if !validStatus(status) {
			status = http.StatusInternalServerError
		}
		message := httpErr.Message
		if message == "" {
			message = http.StatusText(status)
		}
		return Write(ctx, JSON(struct {
			StatusCode int    `json:"statusCode"`
			Message    string `json:"message"`
		}{status, message}).WithStatus(status), status)
	}
	return Write(ctx, JSON(struct {
		StatusCode int    `json:"statusCode"`
		Message    string `json:"message"`
	}{http.StatusInternalServerError, "Internal server error"}).WithStatus(http.StatusInternalServerError), http.StatusInternalServerError)
}

// Writer supports explicit @Res() operations without exposing fasthttp or net/http.
type Writer struct {
	ctx     Transport
	status  int
	written bool
}

func NewWriter(ctx Transport) *Writer               { return &Writer{ctx: ctx} }
func (w *Writer) Status(code int) *Writer           { w.status = code; return w }
func (w *Writer) Header(name, value string) *Writer { w.ctx.SetHeader(name, value); return w }
func (w *Writer) Cookie(cookie Cookie) *Writer {
	_ = appendCookie(w.ctx, cookie)
	return w
}
func (w *Writer) Written() bool   { return w.written }
func (w *Writer) StatusCode() int { return w.status }
func (w *Writer) Write(r Response) error {
	if w.written {
		return errors.New("HTTP response already written")
	}
	if r.StatusCode == 0 {
		r.StatusCode = w.status
	}
	if err := Write(w.ctx, r, w.status); err != nil {
		return err
	}
	w.written = true
	return nil
}
func (w *Writer) JSON(body any) error    { return w.Write(JSON(body)) }
func (w *Writer) Send(body string) error { return w.Write(Text(body)) }
func (w *Writer) End() error {
	return w.Write(Response{StatusCode: w.status, ContentType: "text/plain; charset=utf-8", Body: ""})
}
func (w *Writer) Redirect(url string, status ...int) error { return w.Write(Redirect(url, status...)) }
