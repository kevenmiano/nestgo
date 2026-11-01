package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kevenmiano/nestgo/compiler"
)

func compileCommand(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	var target string
	watch := false
	for _, argument := range args {
		switch argument {
		case "--watch", "-watch":
			watch = true
		default:
			if strings.HasPrefix(argument, "-") || target != "" {
				return fmt.Errorf("usage: nestgo compile [--watch] <directory|file.mgo>")
			}
			target = argument
		}
	}
	if target == "" {
		return fmt.Errorf("usage: nestgo compile [--watch] <directory|file.mgo>")
	}
	info, err := os.Stat(target)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		if !strings.HasSuffix(target, ".mgo") {
			return fmt.Errorf("expected .mgo file")
		}
		target = filepath.Dir(target)
	}
	if !watch {
		if err := compiler.CompileDir(target); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "compiled", target)
		return nil
	}
	fmt.Fprintln(stdout, "watching", target, "(Ctrl+C to stop)")
	return compiler.WatchDir(ctx, target, 250*time.Millisecond, func(err error) {
		if err != nil {
			fmt.Fprintln(stderr, "compile error:", err)
			return
		}
		fmt.Fprintln(stdout, "compiled", target)
	})
}
