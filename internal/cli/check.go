package cli

import (
	"fmt"
	"github.com/kevenmiano/nestgo/compiler"
	"github.com/spf13/cobra"
)

func newCheckCommand() *cobra.Command {
	var format string
	var connections []string
	cmd := &cobra.Command{
		Use:     "check <directory|file.mgo>",
		Short:   "Verify generated artifacts without writing to disk",
		Long:    "Run the NestGo compiler validation pipeline and compare compiler-owned outputs to existing files. Report missing or outdated artifacts without changing source, generated files, or developer-owned test stubs.",
		Args:    cobra.ExactArgs(1),
		Example: "  nestgo compile ./myapp\n  nestgo check ./myapp",
		RunE: func(cmd *cobra.Command, args []string) error {
			if format != "text" && format != "json" {
				return fmt.Errorf("unsupported --format %q (expected text or json)", format)
			}
			dir, err := resolveCompileTarget(args[0])
			if err != nil {
				return finishCompilerCommand(cmd, format, "check", "", err)
			}
			dialects, err := parseConnectionDialects(connections)
			if err != nil { return finishCompilerCommand(cmd, format, "check", "", err) }
			if len(dialects) == 0 { err = compiler.CheckDir(dir) } else { err = compiler.CheckDirWithDialects(dir, dialects) }
			return finishCompilerCommand(cmd, format, "check", "checked "+dir+": generated artifacts up to date", err)
		},
	}
	cmd.Flags().StringVar(&format, "format", "text", "diagnostic output format: text or json")
	cmd.Flags().StringArrayVar(&connections, "connection", nil, "named SQL dialect, repeatable: name:postgres or name:mysql")
	return cmd
}
