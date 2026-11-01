package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kevenmiano/nestgo/compiler"
	"github.com/spf13/cobra"
)

func newCompileCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "compile <directory|file.mgo>",
		Short:   "Compile .mgo decorators into ordinary Go sources and metadata",
		Long:    "Compile all .mgo files in a package directory. Passing a single .mgo file compiles its containing directory.",
		Args:    cobra.ExactArgs(1),
		Example: "  nestgo compile ./examples/shop\n  nestgo compile ./myapp/controllers.mgo",
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
			if err := compiler.CompileDir(path); err != nil {
				return err
			}
			cmd.Printf("compiled %s\n", path)
			return nil
		},
	}
}
