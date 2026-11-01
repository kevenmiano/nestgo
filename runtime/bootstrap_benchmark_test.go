package runtime

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/kevenmiano/nestgo/metadata"
)

// BenchmarkApplicationBootstrapProviders isolates bootstrap startup overhead
// for 1, 10 and 100 injectable provider registrations.
func BenchmarkApplicationBootstrapProviders(b *testing.B) {
	for _, count := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("providers_%d", count), func(b *testing.B) {
			pkg := metadata.Package{}
			for i := 0; i < count; i++ {
				typ := reflect.StructOf([]reflect.StructField{{
					Name: "Marker",
					Type: reflect.TypeOf(0),
					Tag:  reflect.StructTag(fmt.Sprintf("id:%q", fmt.Sprint(i))),
				}})
				pkg.Providers = append(pkg.Providers, metadata.Provider{Type: reflect.PointerTo(typ)})
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := NewApplication(ApplicationOptions{}, pkg); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
