// Package sqlgen converts named SQL parameters into driver bind placeholders.
package sqlgen

import (
	"fmt"
	"strings"
)

type Dialect uint8

const (
	Question Dialect = iota
	PostgreSQL
)

// Bind never interpolates values. It preserves literals and comments and returns
// arguments separately for database/sql or sqlx. The SQL template itself must be trusted.
func Bind(query string, params map[string]any, dialect Dialect) (string, []any, error) {
	if dialect != Question && dialect != PostgreSQL {
		return "", nil, fmt.Errorf("sqlgen: unsupported dialect %d", dialect)
	}
	var out strings.Builder
	args := make([]any, 0)
	used := make(map[string]bool)
	for i := 0; i < len(query); {
		switch {
		case query[i] == '\'':
			j, err := quotedEnd(query, i, '\'', escapeQuotedLiteral(query, i, dialect))
			if err != nil {
				return "", nil, err
			}
			out.WriteString(query[i:j])
			i = j
		case query[i] == '"':
			j, err := quotedEnd(query, i, '"', dialect == Question)
			if err != nil {
				return "", nil, err
			}
			out.WriteString(query[i:j])
			i = j
		case query[i] == '`':
			j, err := quotedEnd(query, i, '`', dialect == Question)
			if err != nil {
				return "", nil, err
			}
			out.WriteString(query[i:j])
			i = j
		case strings.HasPrefix(query[i:], "--") || (dialect == Question && query[i] == '#'):
			j := strings.IndexByte(query[i:], '\n')
			if j < 0 {
				out.WriteString(query[i:])
				i = len(query)
			} else {
				j += i + 1
				out.WriteString(query[i:j])
				i = j
			}
		case strings.HasPrefix(query[i:], "/*"):
			j, err := blockEnd(query, i)
			if err != nil {
				return "", nil, err
			}
			out.WriteString(query[i:j])
			i = j
		case dialect == PostgreSQL && query[i] == '$':
			tag, ok := dollarTag(query[i:])
			if !ok {
				out.WriteByte(query[i])
				i++
				continue
			}
			start := i + len(tag)
			end := strings.Index(query[start:], tag)
			if end < 0 {
				return "", nil, fmt.Errorf("sqlgen: unterminated dollar-quoted literal")
			}
			j := start + end + len(tag)
			out.WriteString(query[i:j])
			i = j
		case strings.HasPrefix(query[i:], "::"):
			out.WriteString("::")
			i += 2
		case query[i] == ':' && i+1 < len(query) && identStart(query[i+1]):
			j := i + 2
			for j < len(query) && identPart(query[j]) {
				j++
			}
			name := query[i+1 : j]
			value, ok := params[name]
			if !ok {
				return "", nil, fmt.Errorf("sqlgen: missing parameter %q", name)
			}
			used[name] = true
			args = append(args, value)
			if dialect == Question {
				out.WriteByte('?')
			} else {
				fmt.Fprintf(&out, "$%d", len(args))
			}
			i = j
		default:
			out.WriteByte(query[i])
			i++
		}
	}
	for name := range params {
		if !used[name] {
			return "", nil, fmt.Errorf("sqlgen: unused parameter %q", name)
		}
	}
	return out.String(), args, nil
}

func identStart(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func identPart(c byte) bool {
	return identStart(c) || c >= '0' && c <= '9'
}

func dollarTag(s string) (string, bool) {
	if len(s) < 2 || s[0] != '$' {
		return "", false
	}
	if s[1] == '$' {
		return "$$", true
	}
	if !identStart(s[1]) {
		return "", false
	}
	for i := 2; i < len(s); i++ {
		if s[i] == '$' {
			return s[:i+1], true
		}
		if !identPart(s[i]) {
			return "", false
		}
	}
	return "", false
}

// escapeQuotedLiteral handles PostgreSQL E'...' and MySQL backslash-escaped strings.
// Standard PostgreSQL strings keep their regular SQL quote rules.
func escapeQuotedLiteral(query string, i int, dialect Dialect) bool {
	if dialect == Question {
		return true
	}
	return i > 0 && (query[i-1] == 'E' || query[i-1] == 'e') && (i == 1 || !identPart(query[i-2]))
}

func quotedEnd(s string, i int, quote byte, escaped bool) (int, error) {
	for j := i + 1; j < len(s); j++ {
		if escaped && s[j] == '\\' && j+1 < len(s) {
			j++
			continue
		}
		if s[j] != quote {
			continue
		}
		if j+1 < len(s) && s[j+1] == quote {
			j++
			continue
		}
		return j + 1, nil
	}
	return 0, fmt.Errorf("sqlgen: unterminated quoted literal")
}

func blockEnd(s string, i int) (int, error) {
	depth := 1
	for j := i + 2; j+1 < len(s); j++ {
		switch {
		case s[j] == '/' && s[j+1] == '*':
			depth++
			j++
		case s[j] == '*' && s[j+1] == '/':
			depth--
			j++
			if depth == 0 {
				return j + 1, nil
			}
		}
	}
	return 0, fmt.Errorf("sqlgen: unterminated block comment")
}
