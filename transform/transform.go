// Package transform provides typed DTO serialization and custom property
// transformation helpers. NestGo-generated JSON methods are honored by
// encoding/json automatically on request input and HTTP response output.
package transform

import (
	"bytes"
	"encoding/json"
)

// Func is the contract for a custom DTO property transformer. Implement one
// with a named function and reference it from @Transform(MyFunction).
type Func[T any] func(T) (T, error)

// Pure adapts an infallible function to a property transformer.
func Pure[T any](fn func(T) T) Func[T] {
	if fn == nil {
		return nil
	}
	return func(value T) (T, error) { return fn(value), nil }
}

// Compose runs transformations in declaration order, stopping at the first error.
func Compose[T any](steps ...Func[T]) Func[T] {
	return func(value T) (T, error) {
		var err error
		for _, step := range steps {
			value, err = ApplyFunc(value, step)
			if err != nil {
				var zero T
				return zero, err
			}
		}
		return value, nil
	}
}

// PlainToInstance constructs T from a plain value (map, slice or raw JSON).
// All conversions pass through encoding/json to respect generated DTO property
// decorators, including those on nested DTOs and DTOs within collections.
// Raw []byte/json.RawMessage inputs are treated as JSON rather than base64.
func PlainToInstance[T any](plain any) (T, error) {
	var value T
	var data []byte
	switch input := plain.(type) {
	case json.RawMessage:
		data = input
	case []byte:
		data = input
	default:
		var err error
		data, err = json.Marshal(plain)
		if err != nil {
			return value, err
		}
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return value, err
	}
	return value, nil
}

// InstanceToPlain returns the JSON-compatible representation of any DTO,
// including slices and nested structures. UseNumber preserves integer precision.
func InstanceToPlain[T any](value T) (any, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var plain any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&plain); err != nil {
		return nil, err
	}
	return plain, nil
}

// Marshal and Unmarshal expose zero-configuration JSON operations for code
// that wants DTO transforms without involving an HTTP adapter.
func Marshal[T any](value T) ([]byte, error)  { return json.Marshal(value) }
func Unmarshal[T any](data []byte) (T, error) { return PlainToInstance[T](data) }
