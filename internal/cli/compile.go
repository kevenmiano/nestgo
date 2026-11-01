package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kevenmiano/nestgo/compiler"
	"github.com/spf13/cobra"
)

func newCompileCommand() *cobra.Command {
	var watch bool
	var interval time.Duration
	var format string
	cmd := &cobra.Command{
		Use:     "compile <directory|file.mgo>",
		Short:   "Compile .mgo decorators into ordinary Go sources and metadata",
		Long:    "Compile all .mgo files in a package directory. Passing a single .mgo file compiles its containing directory. With --watch, compile once, then recompile whenever the input sources change.",
		Args:    cobra.ExactArgs(1),
		Example: "  nestgo compile ./examples/shop\n  nestgo compile --watch ./myapp\n  nestgo compile ./myapp --watch",
		RunE: func(cmd *cobra.Command, args []string) error {
			if format != "text" && format != "json" {
				return fmt.Errorf("unsupported --format %q (expected text or json)", format)
			}
			if watch && format == "json" {
				return finishCompilerCommand(cmd, format, "compile", "", fmt.Errorf("--format=json is only supported for single-shot compile; use text mode with --watch"))
			}
			path, err := resolveCompileTarget(args[0])
			if err != nil {
				return finishCompilerCommand(cmd, format, "compile", "", err)
			}
			if !watch {
				if cmd.Flags().Changed("poll-interval") {
					return finishCompilerCommand(cmd, format, "compile", "", fmt.Errorf("--poll-interval requires --watch"))
				}
				err := compiler.CompileDir(path)
				return finishCompilerCommand(cmd, format, "compile", "compiled "+path, err)
			}
			if interval <= 0 {
				return fmt.Errorf("--poll-interval must be positive")
			}
			cmd.Printf("watching %s (Ctrl+C to stop)\n", path)
			return compiler.WatchDir(cmd.Context(), path, interval, func(err error) {
				if err != nil {
					cmd.PrintErrf("compile error: %v\n", err)
					return
				}
				cmd.Printf("compiled %s\n", path)
			})
		},
	}
	cmd.Flags().StringVar(&format, "format", "text", "diagnostic output format: text or json (single-shot only)")
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "recompile when top-level .mgo sources change")
	cmd.Flags().DurationVar(&interval, "poll-interval", 250*time.Millisecond, "source polling interval in watch mode")
	return cmd
}

func resolveCompileTarget(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("inspect %q: %w", path, err)
	}
	if !info.IsDir() {
		if !strings.HasSuffix(path, ".mgo") {
			return "", fmt.Errorf("expected .mgo file: %q", path)
		}
		return filepath.Dir(path), nil
	}
	return path, nil
}
