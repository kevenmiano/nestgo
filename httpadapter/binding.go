package httpadapter

import (
	"encoding/json"
	"fmt"
	"github.com/kevenmiano/nestgo/validation"
	"reflect"
)

// DecodeBody decodes JSON before applying global, controller and route pipes.
// Generated handlers use this typed path; reflection is not required per call.
func DecodeBody[T any](ctx Context) (T, int, error) {
	var value T
	body := ctx.Body()
	if len(body) > 1<<20 {
		return value, 413, NewHTTPException(413, "request body too large")
	}
	if err := json.Unmarshal(body, &value); err != nil {
		return value, 400, NewHTTPException(400, fmt.Sprintf("invalid JSON: %v", err))
	}
	return TransformBody(ctx, value, 0)
}

// TransformBody permits the compiler to bind a specific argument index.
func TransformBody[T any](ctx Context, value T, index int) (T, int, error) {
	meta := ArgumentMetadata{Index: index, Kind: "Body", Type: reflect.TypeOf((*T)(nil)).Elem()}
	transformed, err := ApplyPipes(ctx, value, meta)
	if err != nil {
		var zero T
		return zero, StatusOf(err), err
	}
	result, ok := transformed.(T)
	if !ok {
		var zero T
		return zero, 400, NewHTTPException(400, "body pipe returned an incompatible type")
	}
	if err := validation.Check(result); err != nil {
		return result, 400, err
	}
	return result, 0, nil
}

// DecodeBodyAt binds JSON at the specified handler argument index.
func DecodeBodyAt[T any](ctx Context, index int) (T, int, error) {
	var value T
	body := ctx.Body()
	if len(body) > 1<<20 {
		return value, 413, NewHTTPException(413, "request body too large")
	}
	if err := json.Unmarshal(body, &value); err != nil {
		return value, 400, NewHTTPException(400, fmt.Sprintf("invalid JSON: %v", err))
	}
	return TransformBody(ctx, value, index)
}
