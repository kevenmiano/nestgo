package http

import stdhttp "net/http"

// HTTPError is a typed error preserving an HTTP status across all adapters.
// Cause is available via errors.Unwrap, but is never serialized to clients.
type HTTPError struct {
	StatusCode int
	Message string
	Cause error
}

func (e *HTTPError) Error() string {
	if e == nil { return "Internal server error" }
	if e.Message != "" { return e.Message }
	if reason := stdhttp.StatusText(e.StatusCode); reason != "" { return reason }
	return "HTTP error"
}
func (e *HTTPError) Unwrap() error {
	if e == nil { return nil }
	return e.Cause
}

// NewHTTPError creates an arbitrary typed HTTP error, including codes not
// represented by convenience helpers.
func NewHTTPError(status int, message string) *HTTPError {
	return &HTTPError{StatusCode:status,Message:message}
}

func newStatusError(status int, message []string) *HTTPError {
	if len(message)==0 || message[0]=="" {
		return NewHTTPError(status, stdhttp.StatusText(status))
	}
	return NewHTTPError(status, message[0])
}

// BadRequest returns an HTTP BadRequest error with an optional custom message.
func BadRequest(message ...string) *HTTPError {
	return newStatusError(StatusBadRequest, message)
}

// Unauthorized returns an HTTP Unauthorized error with an optional custom message.
func Unauthorized(message ...string) *HTTPError {
	return newStatusError(StatusUnauthorized, message)
}

// Forbidden returns an HTTP Forbidden error with an optional custom message.
func Forbidden(message ...string) *HTTPError {
	return newStatusError(StatusForbidden, message)
}

// NotFound returns an HTTP NotFound error with an optional custom message.
func NotFound(message ...string) *HTTPError {
	return newStatusError(StatusNotFound, message)
}

// MethodNotAllowed returns an HTTP MethodNotAllowed error with an optional custom message.
func MethodNotAllowed(message ...string) *HTTPError {
	return newStatusError(StatusMethodNotAllowed, message)
}

// NotAcceptable returns an HTTP NotAcceptable error with an optional custom message.
func NotAcceptable(message ...string) *HTTPError {
	return newStatusError(StatusNotAcceptable, message)
}

// RequestTimeout returns an HTTP RequestTimeout error with an optional custom message.
func RequestTimeout(message ...string) *HTTPError {
	return newStatusError(StatusRequestTimeout, message)
}

// Conflict returns an HTTP Conflict error with an optional custom message.
func Conflict(message ...string) *HTTPError {
	return newStatusError(StatusConflict, message)
}

// Gone returns an HTTP Gone error with an optional custom message.
func Gone(message ...string) *HTTPError {
	return newStatusError(StatusGone, message)
}

// PreconditionFailed returns an HTTP PreconditionFailed error with an optional custom message.
func PreconditionFailed(message ...string) *HTTPError {
	return newStatusError(StatusPreconditionFailed, message)
}

// PayloadTooLarge returns an HTTP PayloadTooLarge error with an optional custom message.
func PayloadTooLarge(message ...string) *HTTPError {
	return newStatusError(StatusPayloadTooLarge, message)
}

// UnsupportedMediaType returns an HTTP UnsupportedMediaType error with an optional custom message.
func UnsupportedMediaType(message ...string) *HTTPError {
	return newStatusError(StatusUnsupportedMediaType, message)
}

// ImATeapot returns an HTTP IAmATeapot error with an optional custom message.
func ImATeapot(message ...string) *HTTPError {
	return newStatusError(StatusIAmATeapot, message)
}

// Misdirected returns an HTTP Misdirected error with an optional custom message.
func Misdirected(message ...string) *HTTPError {
	return newStatusError(StatusMisdirected, message)
}

// UnprocessableEntity returns an HTTP UnprocessableEntity error with an optional custom message.
func UnprocessableEntity(message ...string) *HTTPError {
	return newStatusError(StatusUnprocessableEntity, message)
}

// TooManyRequests returns an HTTP TooManyRequests error with an optional custom message.
func TooManyRequests(message ...string) *HTTPError {
	return newStatusError(StatusTooManyRequests, message)
}

// InternalServerError returns an HTTP InternalServerError error with an optional custom message.
func InternalServerError(message ...string) *HTTPError {
	return newStatusError(StatusInternalServerError, message)
}

// NotImplemented returns an HTTP NotImplemented error with an optional custom message.
func NotImplemented(message ...string) *HTTPError {
	return newStatusError(StatusNotImplemented, message)
}

// BadGateway returns an HTTP BadGateway error with an optional custom message.
func BadGateway(message ...string) *HTTPError {
	return newStatusError(StatusBadGateway, message)
}

// ServiceUnavailable returns an HTTP ServiceUnavailable error with an optional custom message.
func ServiceUnavailable(message ...string) *HTTPError {
	return newStatusError(StatusServiceUnavailable, message)
}

// GatewayTimeout returns an HTTP GatewayTimeout error with an optional custom message.
func GatewayTimeout(message ...string) *HTTPError {
	return newStatusError(StatusGatewayTimeout, message)
}

// HTTPVersionNotSupported returns an HTTP HTTPVersionNotSupported error with an optional custom message.
func HTTPVersionNotSupported(message ...string) *HTTPError {
	return newStatusError(StatusHTTPVersionNotSupported, message)
}

