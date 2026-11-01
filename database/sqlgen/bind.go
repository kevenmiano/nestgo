// Package sqlgen compiles named SQL bind parameters into dialect-specific placeholders.
// It never interpolates parameter values into SQL statements.
package sqlgen

import (
 "fmt"
 "strings"
 "unicode"
)

type Dialect uint8
const (
 Question Dialect = iota
 PostgreSQL
)

// Bind converts :name placeholders to ? or $N and orders arguments by occurrence.
// Parameters inside SQL strings, quoted identifiers, line comments, block comments,
// and PostgreSQL :: casts are not interpreted as bind parameters.
func Bind(query string, params map[string]any, dialect Dialect) (string, []any, error) {
 if dialect != Question && dialect != PostgreSQL { return "", nil, fmt.Errorf("sqlgen: unsupported dialect %d", dialect) }
 var out strings.Builder
 args := make([]any, 0)
 used := make(map[string]bool)
 state := byte(0) // 0 normal, 1 single-quote, 2 double-quote, 3 line comment, 4 block comment, 5 backtick
 for i := 0; i < len(query); {
  c := query[i]
  next := byte(0)
  if i+1 < len(query) { next = query[i+1] }
  switch state {
  case 1,2,5:
   out.WriteByte(c)
   if c == '\\' && state == 1 && i+1 < len(query) { out.WriteByte(next); i+=2; continue }
   quote:=byte('\''); if state==2 { quote='"' }; if state==5 { quote='`' }
   if c==quote {
    if next==quote { out.WriteByte(next); i+=2; continue }
    state=0
   }
   i++
   continue
  case 3:
   out.WriteByte(c); i++
   if c=='\n' { state=0 }
   continue
  case 4:
   out.WriteByte(c); i++
   if c=='*' && next=='/' { out.WriteByte(next); i++; state=0 }
   continue
  }
  if c=='\'' { state=1; out.WriteByte(c); i++; continue }
  if c=='"' { state=2; out.WriteByte(c); i++; continue }
  if c=='`' { state=5; out.WriteByte(c); i++; continue }
  if c=='-' && next=='-' { state=3; out.WriteString("--"); i+=2; continue }
  if c=='/' && next=='*' { state=4; out.WriteString("/*"); i+=2; continue }
  if c==':' && next==':' { out.WriteString("::"); i+=2; continue }
  if c==':' && i+1<len(query) && isStart(rune(next)) {
   j:=i+2
   for j<len(query) && isPart(rune(query[j])) {j++}
   name:=query[i+1:j]
   value,ok:=params[name]
   if !ok { return "",nil,fmt.Errorf("sqlgen: missing parameter %q",name) }
   used[name]=true
   args=append(args,value)
   if dialect==Question {out.WriteByte('?')} else {fmt.Fprintf(&out,"$%d",len(args))}
   i=j
   continue
  }
  out.WriteByte(c);i++
 }
 if state==1 || state==2 || state==4 || state==5 { return "",nil,fmt.Errorf("sqlgen: unterminated SQL literal or comment") }
 for key:=range params { if !used[key] {return "",nil,fmt.Errorf("sqlgen: unused parameter %q",key)} }
 return out.String(),args,nil
}
func isStart(r rune) bool {return r=='_' || unicode.IsLetter(r)}
func isPart(r rune) bool {return isStart(r) || unicode.IsDigit(r)}
