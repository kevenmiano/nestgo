package validation

import (
	"errors"
	"regexp"
	"testing"
)

func TestBuiltIns(t *testing.T) {
	for _, tc := range []struct {
		name string
		want bool
		got  bool
	}{
		{"email", true, IsEmail("ada@example.com")},
		{"display-name address rejected", false, IsEmail("Ada <ada@example.com>")},
		{"email newline", false, IsEmail("ada@example.com\n")},
		{"cpf", true, IsCPF("529.982.247-25")},
		{"cpf repeated", false, IsCPF("111.111.111-11")},
		{"cnpj repeated", false, IsCNPJ("11.111.111/1111-11")},
		{"utf8 minimum", true, MinLength("éç", 2)},
		{"utf8 maximum", false, MaxLength("éç", 1)},
		{"regexp", true, MatchRegexp("ABC123", regexp.MustCompile(`^[A-Z]{3}[0-9]{3}$`))},
		{"uuid", true, IsUUID("123e4567-e89b-42d3-a456-426614174000")},
		{"url", true, IsURL("https://example.com")},
		{"url scheme", false, IsURL("javascript:alert(1)")},
		{"not empty", false, IsNotEmpty("  ")},
	} {
		if tc.got != tc.want {
			t.Errorf("%s: got %v want %v", tc.name, tc.got, tc.want)
		}
	}
}
func TestCustomAndErrors(t *testing.T) {
	if !errors.Is(ApplyFunc("", Func[string](nil)), ErrNilValidator) {
		t.Fatal("nil callback")
	}
	if !errors.Is(Apply[string]("", nil), ErrNilValidator) {
		t.Fatal("nil validator")
	}
	e := Errors{{Field: "email", Rule: "IsEmail", Message: "invalid email"}}
	if e.Error() == "" {
		t.Fatal("missing error")
	}
}
