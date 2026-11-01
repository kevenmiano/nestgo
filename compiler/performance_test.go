package compiler

import (
	"bytes"
	"fmt"
	"testing"
)

func TestGeneratedStaticInvocationWithFallback(t *testing.T) {
	src := []byte("package sample\n@Controller(\"/ready\")\ntype ReadyController struct{}\n@Get(\"/\")\nfunc (c *ReadyController) Ready() string {return \"ready\"}\n@Get(\"/{id}\")\nfunc (c *ReadyController) Find(id string) string{return id}\n")
	file, err := Parse("ready.mgo", src)
	if err != nil {
		t.Fatal(err)
	}
	generated, err := Generate([]*File{file})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(generated, []byte("StaticInvoke:")) {
		t.Fatalf("expected compiled fast path:\n%s", generated)
	}
	if count := bytes.Count(generated, []byte("StaticInvoke:")); count != 1 {
		t.Fatalf("expected exactly one static wrapper, got %d", count)
	}
	if !bytes.Contains(generated, []byte("Handler: \"Find\"")) {
		t.Fatalf("missing dynamic route metadata:\n%s", generated)
	}
}

func BenchmarkCompilerGenerate(b *testing.B) {
	for _, n := range []int{1, 10, 100, 1000} {
		b.Run(fmt.Sprintf("routes_%d", n), func(b *testing.B) {
			var src bytes.Buffer
			src.WriteString("package synthetic\n@Controller(\"/synthetic\")\ntype SyntheticController struct{}\n")
			for i := 0; i < n; i++ {
				fmt.Fprintf(&src, "@Get(\"/%d\")\nfunc (c *SyntheticController) Route%d() string {return \"ok\"}\n", i, i)
			}
			file, err := Parse("synthetic.mgo", src.Bytes())
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := Generate([]*File{file}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkCompilerParseAndGenerate(b *testing.B) {
	for _, n := range []int{1, 10, 100, 1000} {
		b.Run(fmt.Sprintf("routes_%d", n), func(b *testing.B) {
			var src bytes.Buffer
			src.WriteString("package synthetic\n@Controller(\"/synthetic\")\ntype SyntheticController struct{}\n")
			for i := 0; i < n; i++ {
				fmt.Fprintf(&src, "@Get(\"/%d\")\nfunc (c *SyntheticController) Route%d() string {return \"ok\"}\n", i, i)
			}
			input := src.Bytes()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				file, err := Parse("synthetic.mgo", input)
				if err != nil {
					b.Fatal(err)
				}
				if _, err := Generate([]*File{file}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
