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

    @Transform(NormalizeEmail, "toClassOnly")
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
| `@Exclude("toPlainOnly")` | Exclude outgoing JSON only |
| `@Exclude("toClassOnly")` | Ignore incoming JSON only |
| `@Transform(MyFunc)` | Call the custom function on both directions |
| `@Transform(MyFunc, "toClassOnly")` | Transform only on input |
| `@Transform(MyFunc, "toPlainOnly")` | Transform only on output |
| `@Type(AddressDTO)` | Assert the statically declared nested Go type |

Each custom callback has the signature `func(T) (T, error)`,
where `T` is the Go field type. Multiple `@Transform`
decorators execute in declaration order. Local functions and qualified
imported functions are supported (for example
`@Transform(url.PathUnescape, "toClassOnly")`).
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

## Constraints and security

- Decorators occupy separate lines before one exported struct field.
- `@Expose` and `@Exclude` cannot decorate one field.
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
`@Exclude("toPlainOnly")` permits incoming values intentionally.
