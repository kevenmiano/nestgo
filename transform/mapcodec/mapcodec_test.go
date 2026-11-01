package mapcodec

import (
	"testing"
	"time"

	mapstructure "github.com/go-viper/mapstructure/v2"
)

type user struct {
	Name    string        `json:"name"`
	Age     int           `json:"age"`
	Timeout time.Duration `json:"timeout"`
}

func TestDecodeMapAndHooks(t *testing.T) {
	value, err := Decode[user](map[string]any{
		"name":    "Ada",
		"age":     42,
		"timeout": "30s",
	}, WithDecodeHook(mapstructure.StringToTimeDurationHookFunc()), WithStrictFields())
	if err != nil {
		t.Fatal(err)
	}
	if value.Name != "Ada" || value.Age != 42 || value.Timeout != 30*time.Second {
		t.Fatalf("decoded unexpected values: %+v", value)
	}
}

func TestDecodeStrictAndWeakTyping(t *testing.T) {
	_, err := Decode[user](map[string]any{"name": "Ada", "unknown": true}, WithStrictFields())
	if err == nil {
		t.Fatal("strict decoder should reject unknown fields")
	}
	_, err = Decode[user](map[string]any{"age": "42"})
	if err == nil {
		t.Fatal("strict typing should reject numeric strings by default")
	}
	value, err := Decode[user](map[string]any{"age": "42"}, WithWeakTyping())
	if err != nil || value.Age != 42 {
		t.Fatalf("weak conversion: %+v %v", value, err)
	}
}

func BenchmarkDecodeMap(b *testing.B) {
	input := map[string]any{"name": "Ada", "age": 42}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		value, err := Decode[user](input)
		if err != nil || value.Age != 42 {
			b.Fatalf("decode: %+v %v", value, err)
		}
	}
}
