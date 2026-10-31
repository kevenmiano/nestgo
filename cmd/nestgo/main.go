// Command nestgo compiles decorator-annotated .mgo packages into ordinary Go.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kevenmiano/nestgo/compiler"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "diagnose" {
		if err := diagnose(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) != 3 || os.Args[1] != "compile" {
		fmt.Fprintln(os.Stderr, "usage: nestgo compile <directory|file.mgo> | nestgo diagnose -example [-out directory]")
		os.Exit(2)
	}
	path := os.Args[2]
	info, err := os.Stat(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !info.IsDir() {
		if !strings.HasSuffix(path, ".mgo") {
			fmt.Fprintln(os.Stderr, "expected .mgo file")
			os.Exit(2)
		}
		path = filepath.Dir(path)
	}
	if err := compiler.CompileDir(path); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("compiled", path)
}
