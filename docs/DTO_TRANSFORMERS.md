# DTO property transformers

NestGo supports NestJS-inspired property decorators for JSON DTOs. The compiler
generates normal Go methods at build time. Neither HTTP adapter executes a
dynamic decorator interpreter for each request.

## Example

Declare the DTO in a `.mgo` source file:

~~~go
package users

import "strings"

type UserDTO struct {
    @Expose("display_name")
    Name string `json:"name,omitempty"`

    @Exclude()
    Password string `json:"password"`

    @TransformInput(NormalizeEmail)
    Email string `json:"email"`

    @Type(AddressDTO)
    Address AddressDTO `json:"address"`
}

type AddressDTO struct {
    City string `json:"city"`
}

func NormalizeEmail(value string) (string, error) {
    return strings.ToLower(strings.TrimSpace(value)), nil
}
~~~

Run `nestgo compile ./users`. For `users.mgo`, the
compiler writes `users_generated.go` and
`users_transformers_generated.go`. The generated type implements
`json.Marshaler` and `json.Unmarshaler`. The HTTP body
binder and response encoder honor these automatically, including nested DTOs.

## Decorators

| Annotation | Behavior |
| --- | --- |
| `@Expose()` | Include a property with its existing Go JSON name |
| `@Expose("public_name")` | Override the JSON name |
| `@Exclude()` | Exclude incoming and outgoing JSON |
| `@ExcludeOutput()` | Exclude outgoing JSON only |
| `@ExcludeInput()` | Ignore incoming JSON only |
| `@Transform(MyFunc)` | Call the custom function on both directions |
| `@TransformInput(MyFunc)` | Transform only on input |
| `@TransformOutput(MyFunc)` | Transform only on output |
| `@Type(AddressDTO)` | Assert the statically declared nested Go type |

**Direction names are explicit:** `Input` means JSON to DTO (decode), while
`Output` means DTO to JSON (encode). These decorators take no JavaScript-style
direction string options. `@Exclude()` hides a field in both directions,
and `@Transform(MyFunc)` applies the same function in both directions.

Custom function callbacks have signature `func(T) (T, error)`, where `T`
is the Go field type. Struct transformers must satisfy
`transform.Transformer[T]`. Multiple `@Transform`
decorators execute in declaration order. Local functions and qualified
imported functions are supported (for example
`@TransformInput(url.PathUnescape)`).
Transformer failures propagate as JSON encode/decode errors.

Go struct types already determine nested JSON decoding; `@Type`
validates that declaration at compile time rather than invoking a runtime
constructor or coercing arbitrary values.

## Reusable custom transformations

~~~go
import (
    "strings"
    "github.com/kevenmiano/nestgo/transform"
)

var Normalize = transform.Compose(
    transform.Pure(strings.TrimSpace),
    transform.Pure(strings.ToLower),
)
~~~

Then use `@Transform(Normalize)`. There is no mandatory runtime
registration. The `transform` package also exports
`PlainToInstance[T]`, `InstanceToPlain[T]`,
`Marshal[T]` and `Unmarshal[T]` for non-HTTP code.

## Safe custom transformer contracts (functions or structs)

Custom code must satisfy the **Go field type exactly**. It is **not**
necessary to define a struct just for a simple function. For reusable,
state-free struct transformers NestGo exposes a generic interface:

~~~go
type Transformer[T any] interface {
    Transform(value T) (T, error)
}
~~~

An example with both supported forms:

~~~go
// Simple function (no interface implementation needed).
func NormalizeEmail(value string) (string, error) {
    return strings.ToLower(strings.TrimSpace(value)), nil
}

// Struct with a value receiver satisfying transform.Transformer[string].
type Trim struct{}
func (Trim) Transform(value string) (string, error) {
    return strings.TrimSpace(value), nil
}

// Struct with a pointer receiver.
type Mask struct{}
func (*Mask) Transform(value string) (string, error) {
    return "****", nil
}

type AccountDTO struct {
    @TransformInput(NormalizeEmail)
    Email string

    @TransformInput(Trim{})
    FullName string

    @TransformOutput(&Mask{})
    Secret string
}
~~~

The compiler emits calls through
`transform.ApplyFunc[T](value, Fn)` for functions or
`transform.Apply[T](value, Struct{})` for structs. This forces the Go
compiler to verify the argument and return types, including the
`Transform(T) (T, error)` interface. Incorrect signatures, missing
methods, or pointer-receiver mismatches fail during **Go build**, not
during HTTP request dispatch. Struct literals are deliberately limited
to `Type{}` or `&Type{}`; constructors and dynamic expressions
are not evaluated by the decorator compiler.

A nil function, nil interface, or nil step in
`transform.Compose` returns `transform.ErrNilTransformer` instead
of causing a nil-function panic. This guarantee does not prevent
arbitrary panics explicitly written inside user-provided transformers.
Transformers should be stateless or concurrency-safe; instance
injection/configured struct constructors are outside the current
decorator contract.

## Using existing transformation libraries

The HTTP codec uses Go's standard `encoding/json` and generated DTO
methods. This avoids introducing mapstructure reflection in the request path.
`transform.PlainToInstance[T]` accepts JSON bytes, plain maps, slices and
other JSON-serializable values, and always respects DTO JSON hooks (including
nested DTOs).

For dynamic maps, configurations and non-HTTP conversion, the opt-in
`transform/mapcodec` package wraps the maintained
[`go-viper/mapstructure/v2`](https://github.com/go-viper/mapstructure):
```go
import (
    "time"
    "github.com/kevenmiano/nestgo/transform/mapcodec"
    mapstructure "github.com/go-viper/mapstructure/v2"
)

type Settings struct {
    Timeout time.Duration `json:"timeout"`
}

settings, err := mapcodec.Decode[Settings](
    map[string]any{"timeout": "30s"},
    mapcodec.WithDecodeHook(mapstructure.StringToTimeDurationHookFunc()),
    mapcodec.WithStrictFields(),
)
```

The mapcodec adapter also offers `WithWeakTyping()`; it is disabled by
default. **Important:** mapcodec follows mapstructure's own conversion rules
and does not run generated JSON property decorators. For decorated DTOs,
prefer `transform.PlainToInstance` and the regular HTTP JSON pipeline.
This keeps JSON security and transformation behavior consistent while allowing
advanced dynamic decoding only when explicitly requested.

## Constraints and security

- Decorators occupy separate lines before one exported struct field.
- `@Expose` and `@Exclude` cannot decorate one field; `@Exclude` cannot be combined with `@ExcludeInput` or `@ExcludeOutput`.
- Undecorated exported fields honor Go's ordinary JSON tags and options.
- Anonymous embedded fields are not supported in a decorated DTO.
- A DTO cannot declare its own JSON marshal/unmarshal methods alongside
  compiler-generated methods.
- Runtime groups, version filtering, class-level exclusion, discriminator
  support and route-level `@SerializeOptions` are not part of
  this initial implementation.
- Callbacks should be concurrency-safe; encoding or decoding is not a
  substitute for input validation or authorization.

Use `@Exclude()` to keep a field off both JSON boundaries.
`@ExcludeOutput()` permits incoming values intentionally.
