package cli

import (
	"github.com/kevenmiano/nestgo/compiler"
	"github.com/spf13/cobra"
)

func newCheckCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "check <directory|file.mgo>",
		Short:   "Verify generated artifacts without writing to disk",
		Long:    "Run the NestGo compiler validation pipeline and compare compiler-owned outputs to existing files. Report missing or outdated artifacts without changing source, generated files, or developer-owned test stubs.",
		Args:    cobra.ExactArgs(1),
		Example: "  nestgo compile ./myapp\n  nestgo check ./myapp",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := resolveCompileTarget(args[0])
			if err != nil {
				return err
			}
			if err := compiler.CheckDir(dir); err != nil {
				return err
			}
			cmd.Printf("checked %s: generated artifacts up to date\n", dir)
			return nil
		},
	}
}
