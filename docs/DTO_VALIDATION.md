# DTO property validation

NestGo separates **transformation** (which changes a value) from
**validation** (which rejects an invalid value), as NestJS does with
`class-transformer` and `class-validator`. Property validators are
compiled from `.mgo` source into direct Go field checks, without a
runtime decorator interpreter. HTTP input validation is automatic.

## DTO example

```go
package users

type CreateUserDTO struct {
    @IsNotEmpty()
    @MinLength(3)
    Name string `json:"name"`

    @TransformInput(NormalizeEmail)
    @IsEmail()
    Email string `json:"email"`

    @IsCPF()
    CPF string `json:"cpf"`

    @IsCNPJ()
    CNPJ string `json:"cnpj"`

    @Matches("^[A-Z]{3}[0-9]{4}$")
    Code string `json:"code"`

    @Min(18)
    Age int `json:"age"`

    @IsOptional()
    @IsURL()
    Website *string `json:"website,omitempty"`
}
```

When NestGo compiles this DTO, it generates `Validate() error` and
JSON codecs. `httpadapter.DecodeBody` invokes validation after JSON
decoding and input transformations. The reflection-based runtime binder
also checks validation. Invalid input results in HTTP **400**, with a
structured `fields` array. Each error contains a field name, a rule,
and a generic message; raw input values are not echoed.

## Available decorators

| Decorator | Input type | Validation |
| --- | --- | --- |
| `@IsEmail()` | `string` | Email address, not a display name |
| `@IsCPF()`, `@IsCNPJ()` | `string` | Brazilian tax-document format and check digits |
| `@IsCEP()`, `@IsCNH()`, `@IsPIS()` | `string` | Brazilian document checks |
| `@IsPlate()`, `@IsPhone()` | `string` | Brazilian license plate and phone |
| `@Regex("pattern")` / `@Matches("pattern")` | `string` | Go regexp, checked during NestGo compilation |
| `@MinLength(n)`, `@MaxLength(n)`, `@Length(min,max)` | `string` | Unicode code-point length |
| `@IsNotEmpty()` | `string` | Not empty/whitespace-only |
| `@IsURL()`, `@IsUUID()` | `string` | HTTP(S) URL, canonical UUID |
| `@Min(n)`, `@Max(n)` | numeric Go type | Numeric bounds |
| `@IsOptional()` | pointer type | Skip other rules when pointer is nil |
| `@Validate(CustomFn)` | exact Go field type | Custom function returning `error` |

The compiler rejects unsupported decorators, incompatible field types,
malformed parameters and invalid regular expressions. Custom validators
are statically checked by Go through a typed `validation.Func[T]` call.

```go
func MustBeAvailable(name string) error {
    if name == "reserved" {
        return errors.New("unavailable")
    }
    return nil
}

type CreateDTO struct {
    @Validate(MustBeAvailable)
    Name string
}
```

`@IsOptional()` uses a Go pointer, e.g. `*string`, to distinguish
missing/null values from supplied empty values. A supplied empty
string is **not** skipped. With `@IsEmail()`, it is rejected.

To validate non-HTTP values manually, invoke `dto.Validate()` or
`validation.Check(dto)`. No runtime registration is required.

### Libraries and guarantees

CPF/CNPJ and other Brazilian document checks reuse
[brdoc/v4](https://github.com/paemuri/brdoc). Email, URLs, UTF-8
lengths and regexes use Go's standard library. Regular expressions
are validated at compilation and compiled once at application
initialization, rather than once per request.

Validation is not document authenticity verification; CPF/CNPJ
checksums do not confirm registration or ownership. A `Validate()`
method is generated only for DTOs with validation decorators.
Custom callbacks can still return errors or panic if user code
explicitly does so.

This is an initial, intentionally explicit subset of NestJS
`class-validator`; it does not yet claim support for every
decorator, group, conditional validation, nested object collection,
or class-level constraint.
