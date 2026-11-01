package transform

import "errors"

// Transformer is the strongly typed contract for reusable struct-based DTO
// transformations. Implementations can use value or pointer receivers.
//
// The compiler generates calls with the exact DTO field type T, so an
// incompatible Transform signature fails when the application is built.
type Transformer[T any] interface {
	Transform(value T) (T, error)
}

var ErrNilTransformer = errors.New("nestgo: nil DTO transformer")

// Apply invokes a struct-based transformer without reflection or type
// assertions. A zero-valued or configured struct can satisfy Transformer[T].
func Apply[T any](value T, transformer Transformer[T]) (T, error) {
	if transformer == nil {
		var zero T
		return zero, ErrNilTransformer
	}
	return transformer.Transform(value)
}

// ApplyFunc invokes a typed function transformer. A nil callback returns an
// error rather than causing a nil-function panic on an HTTP request.
func ApplyFunc[T any](value T, fn Func[T]) (T, error) {
	if fn == nil {
		var zero T
		return zero, ErrNilTransformer
	}
	return fn(value)
}

// Transform makes the existing Func[T] adapter satisfy Transformer[T].
func (fn Func[T]) Transform(value T) (T, error) {
	return ApplyFunc(value, fn)
}
