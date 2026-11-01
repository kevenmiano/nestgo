// Command nestgo compiles decorator-annotated .mgo packages into ordinary Go.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "diagnose" {
		if err := diagnose(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) < 2 || os.Args[1] != "compile" {
		fmt.Fprintln(os.Stderr, "usage: nestgo compile [--watch] <directory|file.mgo> | nestgo diagnose (-example | -app <go-package>) [-out directory]")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := compileCommand(ctx, os.Args[2:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
