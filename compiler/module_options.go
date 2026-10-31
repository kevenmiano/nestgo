package compiler

import (
	"fmt"
	"go/scanner"
	"go/token"
	"strconv"
	"strings"
)

// moduleOptions models a declarative @Module({imports:[Type], ...}) block.
// Entries are Go type identifiers, NOT string-encoded names or expressions.
type moduleOptions struct {
	Name        string
	Imports     []string
	Providers   []string
	Controllers []string
	Exports     []string
	Defined     map[string]bool
}

type moduleOptionParser struct {
	path string
	line int
	scan scanner.Scanner
	tok  token.Token
	lit  string
}

func newModuleParser(path string, a Annotation) *moduleOptionParser {
	p := &moduleOptionParser{path: path, line: a.Line}
	fset := token.NewFileSet()
	file := fset.AddFile(path, -1, len(a.Argument))
	p.scan.Init(file, []byte(a.Argument), func(_ token.Position, msg string) {
		// Parse errors are reported by expecting the next token; lexer errors
		// (invalid strings) are converted to a diagnostic by the caller.
	}, 0)
	p.next()
	return p
}
func (p *moduleOptionParser) next() { _, p.tok, p.lit = p.scan.Scan() }
func (p *moduleOptionParser) fail(msg string) error {
	return Diagnostic{p.path, p.line, "@Module: " + msg}
}
func (p *moduleOptionParser) expect(want token.Token) error {
	if p.tok != want {
		return p.fail(fmt.Sprintf("expected %s, found %s", want, p.tok))
	}
	p.next()
	return nil
}

func parseModuleOptions(path string, a Annotation, defaultName string) (moduleOptions, error) {
	opts := moduleOptions{Name: defaultName, Defined: map[string]bool{}}
	if !strings.HasPrefix(strings.TrimSpace(a.Argument), "{") {
		if a.Argument == "" {
			return opts, nil
		}
		s, err := strconv.Unquote(a.Argument)
		if err != nil {
			return opts, Diagnostic{path, a.Line, "@Module expects a configuration object, e.g. @Module({imports: [DatabaseModule]})"}
		}
		opts.Name = s
		return opts, nil // old @Module("users") remains source compatible
	}
	p := newModuleParser(path, a)
	if err := p.expect(token.LBRACE); err != nil {
		return opts, err
	}
	for p.tok != token.RBRACE {
		if p.tok == token.EOF {
			return opts, p.fail("unclosed options object")
		}
		if p.tok != token.IDENT && p.tok != token.STRING {
			return opts, p.fail("expected property name")
		}
		key := p.lit
		if p.tok == token.STRING {
			var err error
			key, err = strconv.Unquote(key)
			if err != nil {
				return opts, p.fail("invalid property name")
			}
		}
		p.next()
		if opts.Defined[key] {
			return opts, p.fail("duplicate property " + key)
		}
		opts.Defined[key] = true
		if err := p.expect(token.COLON); err != nil {
			return opts, err
		}
		switch key {
		case "name":
			if p.tok != token.STRING {
				return opts, p.fail("name must be a quoted string")
			}
			name, err := strconv.Unquote(p.lit)
			if err != nil {
				return opts, p.fail("invalid module name")
			}
			opts.Name = name
			p.next()
		case "imports", "providers", "controllers", "exports":
			values, err := p.parseTypeList(key)
			if err != nil {
				return opts, err
			}
			switch key {
			case "imports":
				opts.Imports = values
			case "providers":
				opts.Providers = values
			case "controllers":
				opts.Controllers = values
			case "exports":
				opts.Exports = values
			}
		default:
			return opts, p.fail("unknown property " + key + " (expected name, imports, providers, controllers, exports)")
		}
		if p.tok == token.RBRACE {
			break
		}
		if err := p.expect(token.COMMA); err != nil {
			return opts, err
		}
		if p.tok == token.RBRACE {
			break
		}
	}
	if err := p.expect(token.RBRACE); err != nil {
		return opts, err
	}
	if p.tok != token.EOF {
		return opts, p.fail("unexpected tokens after module options")
	}
	return opts, nil
}
func (p *moduleOptionParser) parseTypeList(field string) ([]string, error) {
	if err := p.expect(token.LBRACK); err != nil {
		return nil, p.fail(field + " must be an array of type references: " + err.Error())
	}
	items := []string{}
	seen := map[string]bool{}
	for p.tok != token.RBRACK {
		if p.tok == token.EOF {
			return nil, p.fail("unclosed " + field + " list")
		}
		if p.tok != token.IDENT {
			return nil, p.fail(field + " requires type identifiers, not quoted strings or expressions")
		}
		value := p.lit
		p.next()
		if p.tok == token.PERIOD {
			return nil, p.fail("qualified types are not yet supported in " + field + "; use local Go type references")
		}
		if seen[value] {
			return nil, p.fail("duplicate type " + value + " in " + field)
		}
		seen[value] = true
		items = append(items, value)
		if p.tok == token.RBRACK {
			break
		}
		if err := p.expect(token.COMMA); err != nil {
			return nil, err
		}
		if p.tok == token.RBRACK {
			break
		}
	}
	if err := p.expect(token.RBRACK); err != nil {
		return nil, err
	}
	// Canonical ordering is required for reproducible generated output.
	sortStrings(items)
	return items, nil
}

func validateModuleOptions(path string, line int, opts moduleOptions, modules, providers, controllers map[string]bool) error {
	for _, group := range []struct {
		names   []string
		kind    string
		allowed map[string]bool
	}{
		{opts.Imports, "module", modules},
		{opts.Providers, "provider", providers},
		{opts.Controllers, "controller", controllers},
		{opts.Exports, "provider", providers},
	} {
		for _, name := range group.names {
			if !group.allowed[name] {
				return Diagnostic{path, line, fmt.Sprintf("@Module: unknown local %s %q", group.kind, name)}
			}
		}
	}
	if opts.Defined["providers"] {
		declared := map[string]bool{}
		for _, name := range opts.Providers {
			declared[name] = true
		}
		for _, name := range opts.Exports {
			if !declared[name] {
				return Diagnostic{path, line, "@Module: exported provider " + name + " must also appear in providers"}
			}
		}
	}
	return nil
}
