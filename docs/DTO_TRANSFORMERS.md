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

## Typed transformer contracts and name-only decorators

For a simple function, use `@TransformInput(NormalizeEmail)`; its
signature must be `func(T) (T, error)`. For a struct, use its **type name**
without allocating or casting it inside the decorator:

~~~go
import (
    "strings"
    "github.com/kevenmiano/nestgo/transform"
)

type Trim struct{}

// Compile-time interface assertion. It never constructs a Trim value.
var _ transform.Transformer[string] = (*Trim)(nil)

func (*Trim) Transform(value string) (string, error) {
    return strings.TrimSpace(value), nil
}

type UserDTO struct {
    @TransformInput(Trim)
    Name string
}
~~~

The declaration-level `var _` assertion is an idiomatic Go compile-time
contract, and can also be used by libraries that implement transformers
outside NestGo. The NestGo compiler additionally emits the equivalent typed
assertion automatically for every decorated field:

~~~go
// Representative generated Go (simplified):
var _ transform.Transformer[string] = (*Trim)(nil)
value, err := transform.Apply[string](input, new(Trim))
~~~

This works for both **value receivers** and **pointer receivers**: the
compiler creates a new, non-nil zero-valued struct for each application.
Named functions also receive a generated type check, equivalent to
`var _ transform.Func[string] = NormalizeEmail`.

Custom struct types must be declared in a `.mgo` file in the same Go package,
or in an ordinary `.go` file in that package. Functions may be local or
package-qualified, as before. Dynamic constructors or configured instances
are intentionally not accepted in decorators: `@TransformInput(Trim{})`
and `@TransformInput(&Trim{})` are rejected. Use a typed function or a
separate provider-based mechanism when configuration is required.

Incorrect signatures, missing methods, and incompatible field types fail
during Go compilation, not at the first HTTP request. Nil callbacks or
composed nil steps return `transform.ErrNilTransformer` rather than
invoking a nil function. These guarantees do not prevent a panic deliberately
written in custom transformation code. Implementations should be stateless
or concurrency-safe.

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
