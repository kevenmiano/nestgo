package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kevenmiano/nestgo/compiler"
	"github.com/kevenmiano/nestgo/database/sqlgen"
)

func main() {
	if len(os.Args) < 3 {
		usage()
	}
	if os.Args[1] == "compile-repository" {
		if len(os.Args) != 4 {
			usage()
		}
		var dialect sqlgen.Dialect
		switch os.Args[3] {
		case "postgres":
			dialect = sqlgen.PostgreSQL
		case "mysql":
			dialect = sqlgen.Question
		default:
			fmt.Fprintln(os.Stderr, "expected postgres or mysql dialect")
			os.Exit(2)
		}
		path, err := compiler.CompileRepositoryFile(os.Args[2], dialect)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("generated", path)
		return
	}
	if os.Args[1] != "compile" || len(os.Args) < 3 {
		usage()
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
	dialects := make(map[string]sqlgen.Dialect)
	for _, flag := range os.Args[3:] {
		if !strings.HasPrefix(flag, "--connection=") {
			usage()
		}
		name, dialectName, ok := strings.Cut(strings.TrimPrefix(flag, "--connection="), ":")
		if !ok || name == "" {
			fmt.Fprintln(os.Stderr, "expected --connection=<name>:<postgres|mysql>")
			os.Exit(2)
		}
		if _, exists := dialects[name]; exists {
			fmt.Fprintln(os.Stderr, "duplicate connection dialect:", name)
			os.Exit(2)
		}
		switch dialectName {
		case "postgres":
			dialects[name] = sqlgen.PostgreSQL
		case "mysql":
			dialects[name] = sqlgen.Question
		default:
			fmt.Fprintln(os.Stderr, "unsupported dialect for connection:", name)
			os.Exit(2)
		}
	}
	var err error
	if len(dialects) > 0 {
		err = compiler.CompileDirWithDialects(path, dialects)
	} else {
		err = compiler.CompileDir(path)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("compiled", path)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: modforge compile <directory|file.mgo> [--connection=<name>:<postgres|mysql> ...] | modforge compile-repository <file.mgo> <postgres|mysql>")
	os.Exit(2)
}
