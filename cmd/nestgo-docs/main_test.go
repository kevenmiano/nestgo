package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareForDocusaurus(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "nestgo_completion_bash.md")
	content := "Run source <(nestgo completion bash) and ${fpath[1]}\n\n```bash\nsource <(nestgo completion bash) and ${fpath[1]}\n```\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := prepareForDocusaurus(directory); err != nil {
		t.Fatal(err)
	}
	result, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result), "source &lt;(nestgo completion bash) and $&#123;fpath[1]&#125;") {
		t.Fatalf("non-fenced shell syntax was not escaped: %s", result)
	}
	if !strings.Contains(string(result), "```bash\nsource <(nestgo completion bash) and ${fpath[1]}") {
		t.Fatalf("fenced examples must remain unchanged: %s", result)
	}
}
