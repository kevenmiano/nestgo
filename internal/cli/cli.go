// Package cli defines the NestGo command tree. Each invocation creates fresh commands,
// so tests and documentation generation do not share mutable flag state.
package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

// Version is injected by GoReleaser for published CLI binaries.
var Version = "dev"

// New creates the installable nestgo command and its subcommands.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:           "nestgo",
		Short:         "NestGo framework developer tools",
		Long:          "Compile NestGo decorators and inspect application dependency graphs and startup performance.",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	root.AddCommand(newCompileCommand(), newDiagnoseCommand())
	return root
}

// Execute runs the command tree while retaining the diagnostic flags previously
// accepted as single-dash Go flags (for example -example and -cold-samples).
// Cobra's native spelling with two dashes remains available.
func Execute(args []string) error {
	cmd := New()
	cmd.SetArgs(normalizeArgs(args))
	return cmd.Execute()
}

func normalizeArgs(args []string) []string {
	out := append([]string(nil), args...)
	legacy := map[string]bool{
		"example": true, "app": true, "out": true, "samples": true,
		"iterations": true, "cold-samples": true, "scale": true,
		"cold-child": true, "strict": true,
	}
	diagnose := false
	for i, arg := range out {
		if arg == "--" {
			break
		}
		if arg == "diagnose" {
			diagnose = true
			continue
		}
		if !diagnose || !strings.HasPrefix(arg, "-") || strings.HasPrefix(arg, "--") {
			continue
		}
		flagName := strings.TrimPrefix(arg, "-")
		if name, _, found := strings.Cut(flagName, "="); found {
			flagName = name
		}
		if legacy[flagName] {
			out[i] = "-" + arg
		}
	}
	return out
}
