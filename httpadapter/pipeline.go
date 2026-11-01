package httpadapter

import (
	"errors"
	"fmt"
	"reflect"
)

// ExecutionContext exposes the HTTP request and selected controller to guards
// and exception filters without leaking transport-specific request objects.
type ExecutionContext struct {
	Request    Context
	Controller any
	Handler    string
}

// Next continues the middleware chain. Middleware may stop processing by not
// invoking Next, or return an error to the global exception filters.
type Next func() error
type Middleware func(Context, Next) error

// Guard runs before parameter binding and handler invocation.
type Guard interface {
	CanActivate(ExecutionContext) (bool, error)
}

// ArgumentMetadata describes the binding being transformed.
type ArgumentMetadata struct {
	Index int
	Kind  string
	Name  string
	Type  reflect.Type
}

// Pipe validates or transforms a single bound argument. It must return a value
// assignable to the handler parameter's Go type.
type Pipe interface {
	Transform(any, ArgumentMetadata) (any, error)
}

// ExceptionFilter may handle an error by writing to the HTTP context and
// returning (true, nil). Returning false delegates to the next filter.
type ExceptionFilter interface {
	Catch(error, ExecutionContext) (handled bool, writeErr error)
}

// HTTPException marks errors whose status and message are safe for clients.
// Internal failures deliberately do not expose their underlying error text.
type HTTPException struct {
	Status  int
	Message string
}

func (e *HTTPException) Error() string {
	if e == nil {
		return "HTTP exception"
	}
	return e.Message
}
func NewHTTPException(status int, message string) *HTTPException {
	return &HTTPException{Status: status, Message: message}
}

func StatusOf(err error) int {
	var exception *HTTPException
	if errors.As(err, &exception) && exception != nil && exception.Status >= 400 && exception.Status <= 599 {
		return exception.Status
	}
	return 500
}

func WriteError(ctx Context, err error) error {
	status := StatusOf(err)
	message := "Internal server error"
	if status < 500 {
		message = err.Error()
		if message == "" {
			message = "Bad Request"
		}
	}
	ctx.Status(status)
	ctx.SetHeader("Content-Type", "text/plain; charset=utf-8")
	return ctx.Write([]byte(message + "\n"))
}

// PipeApplier is an optional capability implemented by the request-scoped
// context wrapper. Adapters remain unaware of framework pipes.
type PipeApplier interface {
	ApplyPipes(any, ArgumentMetadata) (any, error)
}

func ApplyPipes(ctx Context, value any, meta ArgumentMetadata) (any, error) {
	if applier, ok := ctx.(PipeApplier); ok {
		return applier.ApplyPipes(value, meta)
	}
	return value, nil
}

func bindString(ctx Context, value string, kind, name string, index int) (string, int, error) {
	result, err := ApplyPipes(ctx, value, ArgumentMetadata{Index: index, Kind: kind, Name: name, Type: reflect.TypeOf("")})
	if err != nil {
		return "", StatusOf(err), err
	}
	s, ok := result.(string)
	if !ok {
		return "", 400, NewHTTPException(400, fmt.Sprintf("%s pipe must return string", kind))
	}
	return s, 0, nil
}

func BindParam(ctx Context, name string, index int) (string, int, error) {
	return bindString(ctx, ctx.Param(name), "Param", name, index)
}
func BindQuery(ctx Context, name string, index int) (string, int, error) {
	return bindString(ctx, ctx.Query(name), "Query", name, index)
}
