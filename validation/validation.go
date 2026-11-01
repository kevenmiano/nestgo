// Package validation supplies typed validators for NestGo DTOs.
package validation

import (
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/paemuri/brdoc/v4"
)

// FieldError describes a failed DTO constraint without exposing input values.
type FieldError struct {
	Field   string `json:"field"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}
type Errors []FieldError

func (e Errors) Error() string {
	if len(e) == 0 {
		return "validation failed"
	}
	return fmt.Sprintf("validation failed: %s (%s)", e[0].Field, e[0].Rule)
}

type Func[T any] func(T) error
type Validator[T any] interface{ Validate(T) error }

var ErrNilValidator = errors.New("nestgo: nil validator")

func ApplyFunc[T any](value T, fn Func[T]) error {
	if fn == nil {
		return ErrNilValidator
	}
	return fn(value)
}
func Apply[T any](value T, validator Validator[T]) error {
	if validator == nil {
		return ErrNilValidator
	}
	return validator.Validate(value)
}

// Check invokes a compiled DTO's Validate method when available.
func Check(value any) error {
	if v, ok := value.(interface{ Validate() error }); ok {
		return v.Validate()
	}
	return nil
}

func IsEmail(value string) bool {
	if value == "" || strings.TrimSpace(value) != value || strings.ContainsAny(value, "\r\n") {
		return false
	}
	addr, err := mail.ParseAddress(value)
	return err == nil && addr.Address == value && strings.Count(value, "@") == 1
}
func IsCPF(value string) bool                          { return brdoc.IsCPF(value) }
func IsCNPJ(value string) bool                         { return brdoc.IsCNPJ(value) }
func IsCEP(value string) bool                          { valid, _ := brdoc.IsCEP(value); return valid }
func IsCNH(value string) bool                          { return brdoc.IsCNH(value) }
func IsPIS(value string) bool                          { return brdoc.IsPIS(value) }
func IsPlate(value string) bool                        { return brdoc.IsPlate(value) }
func IsPhone(value string) bool                        { valid, _ := brdoc.IsPhone(value); return valid }
func IsNotEmpty(value string) bool                     { return strings.TrimSpace(value) != "" }
func MinLength(value string, min int) bool             { return utf8.RuneCountInString(value) >= min }
func MaxLength(value string, max int) bool             { return utf8.RuneCountInString(value) <= max }
func MatchRegexp(value string, re *regexp.Regexp) bool { return re != nil && re.MatchString(value) }
func IsURL(value string) bool {
	u, err := url.ParseRequestURI(value)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != ""
}

var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func IsUUID(value string) bool { return uuidPattern.MatchString(value) }
