package compiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

// literalDecoratorArgs only accepts statically known string/numeric arguments.
func literalDecoratorArgs(argument string) ([]*ast.BasicLit, error) {
	expr, err := parser.ParseExpr("[]any{" + argument + "}")
	if err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	lit, ok := expr.(*ast.CompositeLit)
	if !ok {
		return nil, fmt.Errorf("invalid argument list")
	}
	result := make([]*ast.BasicLit, 0, len(lit.Elts))
	for _, el := range lit.Elts {
		v, ok := el.(*ast.BasicLit)
		if !ok {
			return nil, fmt.Errorf("arguments must be literals")
		}
		result = append(result, v)
	}
	return result, nil
}

func parseHeaderDecorator(argument string) (string, string, error) {
	args, err := literalDecoratorArgs(argument)
	if err != nil {
		return "", "", err
	}
	if len(args) != 2 || args[0].Kind != token.STRING || args[1].Kind != token.STRING {
		return "", "", fmt.Errorf("expected two quoted strings: name, value")
	}
	name, err := strconv.Unquote(args[0].Value)
	if err != nil {
		return "", "", err
	}
	value, err := strconv.Unquote(args[1].Value)
	if err != nil {
		return "", "", err
	}
	if name == "" || strings.ContainsAny(name+value, "\r\n") {
		return "", "", fmt.Errorf("invalid header name or value")
	}
	return name, value, nil
}

func parseRedirectDecorator(argument string) (string, int, error) {
	args, err := literalDecoratorArgs(argument)
	if err != nil {
		return "", 0, err
	}
	if len(args) < 1 || len(args) > 2 || args[0].Kind != token.STRING {
		return "", 0, fmt.Errorf("expected quoted URL and optional 3xx status")
	}
	url, err := strconv.Unquote(args[0].Value)
	if err != nil {
		return "", 0, err
	}
	if url == "" || strings.ContainsAny(url, "\r\n") {
		return "", 0, fmt.Errorf("invalid redirect URL")
	}
	status := 302
	if len(args) == 2 {
		if args[1].Kind != token.INT {
			return "", 0, fmt.Errorf("redirect status must be an integer")
		}
		status, err = strconv.Atoi(args[1].Value)
		if err != nil || status < 300 || status > 399 {
			return "", 0, fmt.Errorf("redirect status must be 3xx")
		}
	}
	return url, status, nil
}
