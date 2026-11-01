package transform

import (
	"errors"
	"strings"
	"testing"
)

type Trim struct{}

func (Trim) Transform(value string) (string, error) {
	return strings.TrimSpace(value), nil
}

type Uppercase struct{}

func (*Uppercase) Transform(value string) (string, error) {
	return strings.ToUpper(value), nil
}

var _ Transformer[string] = Trim{}
var _ Transformer[string] = (*Uppercase)(nil)
var _ Transformer[string] = Func[string](nil)

func TestTypedStructAndFunctionTransformers(t *testing.T) {
	first, err := Apply("  ada  ", Trim{})
	if err != nil || first != "ada" {
		t.Fatalf("value receiver failed: %q %v", first, err)
	}
	second, err := Apply(first, &Uppercase{})
	if err != nil || second != "ADA" {
		t.Fatalf("pointer receiver failed: %q %v", second, err)
	}
	third, err := ApplyFunc(second, Func[string](func(value string) (string, error) {
		return value + "!", nil
	}))
	if err != nil || third != "ADA!" {
		t.Fatalf("function callback failed: %q %v", third, err)
	}
}

func TestNilCallbacksReturnErrorsInsteadOfPanicking(t *testing.T) {
	var missing Func[string]
	if _, err := ApplyFunc("input", missing); !errors.Is(err, ErrNilTransformer) {
		t.Fatalf("nil function returned %v", err)
	}
	if _, err := missing.Transform("input"); !errors.Is(err, ErrNilTransformer) {
		t.Fatalf("nil function adapter returned %v", err)
	}
	if _, err := Apply[string]("input", nil); !errors.Is(err, ErrNilTransformer) {
		t.Fatalf("nil struct interface returned %v", err)
	}
	if _, err := Compose(missing, Pure(strings.TrimSpace))("input"); !errors.Is(err, ErrNilTransformer) {
		t.Fatalf("nil Compose step returned %v", err)
	}
	if _, err := Compose(Pure[string](nil))("input"); !errors.Is(err, ErrNilTransformer) {
		t.Fatalf("nil Pure function returned %v", err)
	}
}

func BenchmarkTypedTransformerInterface(b *testing.B) {
	var result string
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		value, err := Apply("  ada  ", Trim{})
		if err != nil {
			b.Fatal(err)
		}
		result = value
	}
	_ = result
}
