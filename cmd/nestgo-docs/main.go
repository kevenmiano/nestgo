// Command nestgo-docs generates the CLI reference directly from the Cobra command tree.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kevenmiano/nestgo/internal/cli"
	"github.com/spf13/cobra/doc"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: nestgo-docs <output-directory>")
		os.Exit(2)
	}
	root := cli.New()
	root.DisableAutoGenTag = true
	root.InitDefaultCompletionCmd()
	root.InitDefaultHelpCmd()
	if err := os.MkdirAll(os.Args[1], 0755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := doc.GenMarkdownTree(root, os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := prepareForDocusaurus(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// prepareForDocusaurus escapes bash/zsh process substitutions from Cobra help.
// Unescaped "<(" in prose is interpreted by Docusaurus MDX as malformed JSX.\n// Shell ${...} expansions must also be escaped to prevent MDX JS evaluation.
func prepareForDocusaurus(directory string) error {
	pages, err := filepath.Glob(filepath.Join(directory, "*.md"))
	if err != nil {
		return err
	}
	for _, path := range pages {
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines := strings.Split(string(raw), "\n")
		inFence := false
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
				inFence = !inFence
				continue
			}
			if !inFence {
				lines[i] = strings.NewReplacer("<(", "&lt;(", "{", "&#123;", "}", "&#125;").Replace(line)
			}
		}
		if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644); err != nil {
			return err
		}
	}
	return nil
}
