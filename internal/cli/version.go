package cli

import "github.com/spf13/cobra"

// newVersionCommand exposes the same build metadata as the root --version flag.
func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the NestGo CLI version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println(Version)
			return nil
		},
	}
}
