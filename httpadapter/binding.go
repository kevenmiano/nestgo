package httpadapter

import (
	"encoding/json"
	"fmt"
	"github.com/kevenmiano/nestgo/validation"
)

// DecodeBody reads a JSON request into a concrete Go type without reflection
// based handler invocation. The limit matches the dynamic runtime binder.
func DecodeBody[T any](ctx Context) (T, int, error) {
	var value T
	body := ctx.Body()
	if len(body) > 1<<20 {
		return value, 413, fmt.Errorf("request body too large")
	}
	if err := json.Unmarshal(body, &value); err != nil {
		return value, 400, fmt.Errorf("invalid JSON: %w", err)
	}
	if err := validation.Check(value); err != nil {
		return value, 400, err
	}
	return value, 0, nil
}
