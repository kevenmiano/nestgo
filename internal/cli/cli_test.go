package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra/doc"
)

func executeForTest(args ...string) (string, error) {
	var output bytes.Buffer
	cmd := New()
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs(normalizeArgs(args))
	err := cmd.Execute()
	return output.String(), err
}

func TestRootAndSubcommandHelp(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want []string
	}{
		{[]string{"--help"}, []string{"compile", "diagnose", "completion"}},
		{[]string{"compile", "--help"}, []string{"<directory|file.mgo>", "Compile"}},
		{[]string{"diagnose", "--help"}, []string{"--app", "--cold-samples", "--strict"}},
		{[]string{"completion", "--help"}, []string{"bash", "zsh", "fish", "powershell"}},
		{[]string{"--version"}, []string{"dev"}},
	} {
		output, err := executeForTest(tc.args...)
		if err != nil {
			t.Fatalf("%v: %v", tc.args, err)
		}
		for _, want := range tc.want {
			if !strings.Contains(output, want) {
				t.Errorf("%v: expected %q in %q", tc.args, want, output)
			}
		}
	}
}

func TestInvalidCommandsAndArguments(t *testing.T) {
	for _, args := range [][]string{
		{"not-a-command"},
		{"compile"},
		{"compile", "./a.mgo", "./b.mgo"},
		{"compile", "does-not-exist.mgo"},
		{"diagnose", "-example", "unexpected"},
		{"diagnose", "-cold-samples", "-1", "-example"},
		{"diagnose", "-example", "-app", "./examples/shop"},
		{"diagnose"},
	} {
		_, err := executeForTest(args...)
		if err == nil {
			t.Errorf("expected error for %v", args)
		}
	}
}

func TestDiagnosticSampleProducesReports(t *testing.T) {
	out := filepath.Join(t.TempDir(), "report")
	_, err := executeForTest("diagnose", "-example", "-samples", "1", "-iterations", "1", "-cold-samples", "0", "-out", out)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"report.html", "report.json", "graph.json", "modules.json", "graph.dot", "modules.dot"} {
		info, err := os.Stat(filepath.Join(out, name))
		if err != nil || info.Size() == 0 {
			t.Errorf("missing or empty %s: %v", name, err)
		}
	}
}

func TestGeneratedCLIReference(t *testing.T) {
	root := New()
	root.DisableAutoGenTag = true
	root.InitDefaultCompletionCmd()
	root.InitDefaultHelpCmd()
	directory := t.TempDir()
	if err := doc.GenMarkdownTree(root, directory); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"nestgo.md", "nestgo_compile.md", "nestgo_diagnose.md", "nestgo_completion.md"} {
		contents, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil || !strings.Contains(string(contents), "nestgo") {
			t.Errorf("missing CLI reference page %s: %v", name, err)
		}
	}
}

func TestPrivateFlagsStayHidden(t *testing.T) {
	output, err := executeForTest("diagnose", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output, "cold-child") {
		t.Fatal("internal diagnostic benchmark child flag must not be documented")
	}
}

func TestSingleDashDiagnosticFlagsRemainAccepted(t *testing.T) {
	out := normalizeArgs([]string{"diagnose", "-example", "-cold-samples=0", "-strict", "--", "-app"})
	want := []string{"diagnose", "--example", "--cold-samples=0", "--strict", "--", "-app"}
	for i, value := range want {
		if out[i] != value {
			t.Fatalf("argument %d: got %q, want %q", i, out[i], value)
		}
	}
	if _, err := executeForTest("diagnose", "-example", "-samples=1", "-iterations=1", "-cold-samples=0", "-out", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if _, err := executeForTest("diagnose", "--example", "--samples=1", "--iterations=1", "--cold-samples=0", "--out", t.TempDir()); err != nil {
		t.Fatal(err)
	}
}
