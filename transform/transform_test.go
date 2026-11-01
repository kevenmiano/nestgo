package transform

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type simple struct { Name string `json:"name"` }

func TestPlainInstanceRoundTrip(t *testing.T) {
	value, err := PlainToInstance[simple]([]byte(`{"name":"Ada"}`))
	if err != nil || value.Name != "Ada" { t.Fatalf("decode: %+v %v", value, err) }
	plain, err := InstanceToPlain([]simple{value})
	if err != nil { t.Fatal(err) }
	records, ok := plain.([]any)
	if !ok || len(records) != 1 || records[0].(map[string]any)["name"] != "Ada" {
		t.Fatalf("plain: %#v", plain)
	}
	raw, err := Marshal(value)
	if err != nil || !json.Valid(raw) { t.Fatalf("marshal: %s %v", raw, err) }
}

func TestComposeAndErrors(t *testing.T) {
	lower := Pure(strings.ToLower)
	fn := Compose(lower, Func[string](func(v string) (string,error) { return v+"!", nil }))
	result, err := fn("ADA")
	if err != nil || result != "ada!" { t.Fatalf("%q: %v",result,err) }
	boom := errors.New("bad")
	failing := Compose(lower, Func[string](func(string) (string,error) { return "",boom }))
	if _, err := failing("test"); !errors.Is(err,boom) { t.Fatalf("got %v",err) }
}

func BenchmarkPlainToInstance(b *testing.B) {
	src := []byte(`{"name":"Ada"}`)
	b.ReportAllocs()
	for i:=0;i<b.N;i++ {
		if _,err := PlainToInstance[simple](src);err!=nil {b.Fatal(err)}
	}
}
