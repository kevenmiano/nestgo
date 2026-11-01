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
	cmd := &cobra.Command{
		Use:   "compile <directory|file.mgo>",
		Short: "Compile .mgo decorators into ordinary Go sources and metadata",
		Long: "Compile all .mgo files in a package directory. Passing a single .mgo file compiles its containing directory. With --watch, compile once, then recompile whenever the input sources change.",
		Args:  cobra.ExactArgs(1),
		Example: "  nestgo compile ./examples/shop\n  nestgo compile --watch ./myapp\n  nestgo compile ./myapp --watch",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			info, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("inspect %q: %w", path, err)
			}
			if !info.IsDir() {
				if !strings.HasSuffix(path, ".mgo") {
					return fmt.Errorf("expected .mgo file: %q", path)
				}
				path = filepath.Dir(path)
			}
			if !watch {
				if cmd.Flags().Changed("poll-interval") {
					return fmt.Errorf("--poll-interval requires --watch")
				}
				if err := compiler.CompileDir(path); err != nil {
					return err
				}
				cmd.Printf("compiled %s\n", path)
				return nil
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
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "recompile when top-level .mgo sources change")
	cmd.Flags().DurationVar(&interval, "poll-interval", 250*time.Millisecond, "source polling interval in watch mode")
	return cmd
}
