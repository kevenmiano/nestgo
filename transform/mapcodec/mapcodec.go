// Package mapcodec provides an opt-in mapstructure/v2 adapter for converting
// arbitrary maps to Go DTOs. The default NestGo HTTP JSON path continues to use
// generated typed JSON methods instead of reflection-based map conversion.
package mapcodec

import mapstructure "github.com/go-viper/mapstructure/v2"

// Option customizes map decoding without exposing DecoderConfig to consumers.
type Option func(*mapstructure.DecoderConfig)

// WithDecodeHook applies a mapstructure decode hook. The underlying library
// supports primitive, map, and collection conversions and hook composition.
func WithDecodeHook(hook mapstructure.DecodeHookFunc) Option {
	return func(config *mapstructure.DecoderConfig) {
		if config.DecodeHook == nil {
			config.DecodeHook = hook
		} else {
			config.DecodeHook = mapstructure.ComposeDecodeHookFunc(config.DecodeHook, hook)
		}
	}
}

// WithWeakTyping allows type coercion (e.g., string "42" to integer 42).
// Disabled by default to avoid surprising or unsafe input conversion.
func WithWeakTyping() Option {
	return func(config *mapstructure.DecoderConfig) {
		config.WeaklyTypedInput = true
	}
}

// WithStrictFields rejects keys absent from the target struct.
func WithStrictFields() Option {
	return func(config *mapstructure.DecoderConfig) {
		config.ErrorUnused = true
	}
}

// Decode converts a map or other mapstructure-supported value into T.
//
// Note: For DTOs with NestGo property decorators, prefer transform.PlainToInstance
// (which respects generated JSON methods). Decode deliberately uses mapstructure's
// own map semantics and decode hooks and does not invoke generated JSON methods.
func Decode[T any](plain any, options ...Option) (T, error) {
	var result T
	config := &mapstructure.DecoderConfig{
		Result:           &result,
		TagName:          "json",
		WeaklyTypedInput: false,
		ZeroFields:       true,
	}
	for _, option := range options {
		if option != nil {
			option(config)
		}
	}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return result, err
	}
	if err := decoder.Decode(plain); err != nil {
		return result, err
	}
	return result, nil
}
