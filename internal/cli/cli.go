// Package cli defines the NestGo command tree. Each invocation creates fresh commands,
// so tests and documentation generation do not share mutable flag state.
package cli

import "github.com/spf13/cobra"

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
