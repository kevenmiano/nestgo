// Command nestgo provides the NestGo developer CLI.
package main

import (
	"fmt"
	"os"

	"github.com/kevenmiano/nestgo/internal/cli"
)

func main() {
	if err := cli.New().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
