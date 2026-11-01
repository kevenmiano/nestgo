// Command nestgo-docs generates the CLI reference directly from the Cobra command tree.
package main

import (
	"fmt"
	"os"

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
}
