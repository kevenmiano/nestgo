// Package diagnostics provides reproducible dependency graphs and runtime startup measurements.
// Structural output is deterministic; startup timing is measured, not predicted.
package diagnostics

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/kevenmiano/nestgo/metadata"
)

type Node struct {
	ID string `json:"id"`
}
type Edge struct {
	From string `json:"from"`
	To string `json:"to"`
	Field string `json:"field"`
}
type Finding struct {
	Kind string `json:"kind"`
	Provider string `json:"provider"`
	Field string `json:"field,omitempty"`
	Details string `json:"details"`
}
type Graph struct {
	SchemaVersion int `json:"schemaVersion"`
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
	Cycles [][]string `json:"cycles"`
	Findings []Finding `json:"findings"`
}
func typeID(t reflect.Type) string {
	for t.Kind() == reflect.Ptr { t = t.Elem() }
	if t.PkgPath() == "" { return t.String() }
	if t.Name() == "" { return t.String() }
	return t.PkgPath() + "." + t.Name()
}

// Analyze examines the dependency relationships that the current NestGo runtime
// can inject (exported pointer and interface fields), without constructing services.
// It does not yet model module imports/exports or future provider scopes.
func Analyze(packages ...metadata.Package) Graph {
	g := Graph{SchemaVersion: 1, Nodes: []Node{}, Edges: []Edge{}, Cycles: [][]string{}, Findings: []Finding{}}
	providers := map[reflect.Type]bool{}
	for _, pkg := range packages {
		for _, p := range pkg.Providers {
			if p.Type == nil || p.Type.Kind() != reflect.Ptr || p.Type.Elem().Kind() != reflect.Struct {
				g.Findings = append(g.Findings, Finding{Kind: "invalid-provider", Provider: fmt.Sprint(p.Type), Details: "provider must be pointer to struct"})
				continue
			}
			if providers[p.Type] { g.Findings = append(g.Findings, Finding{Kind: "duplicate-provider", Provider: typeID(p.Type), Details: "provider is registered more than once"}) }
			providers[p.Type] = true
		}
	}
	types := make([]reflect.Type, 0, len(providers))
	for t := range providers { types = append(types, t) }
	sort.Slice(types, func(i, j int) bool { return typeID(types[i]) < typeID(types[j]) })
	for _, t := range types { g.Nodes = append(g.Nodes, Node{ID: typeID(t)}) }
	for _, t := range types {
		source := typeID(t)
		for i := 0; i < t.Elem().NumField(); i++ {
			field := t.Elem().Field(i)
			// Keep exact parity with the current runtime injection rules.
			if field.Anonymous || !field.IsExported() || (field.Type.Kind() != reflect.Ptr && field.Type.Kind() != reflect.Interface) { continue }
			var targets []reflect.Type
			if providers[field.Type] { targets = append(targets, field.Type) }
			if len(targets) == 0 && field.Type.Kind() == reflect.Interface {
				for _, p := range types { if p.Implements(field.Type) { targets = append(targets, p) } }
			}
			switch len(targets) {
			case 0:
				g.Findings = append(g.Findings, Finding{Kind: "missing-provider", Provider: source, Field: field.Name, Details: "no registered provider satisfies " + field.Type.String()})
			case 1:
				g.Edges = append(g.Edges, Edge{From: source, To: typeID(targets[0]), Field: field.Name})
			default:
				names := make([]string, 0, len(targets))
				for _, target := range targets { names = append(names, typeID(target)) }
				sort.Strings(names)
				g.Findings = append(g.Findings, Finding{Kind: "ambiguous-provider", Provider: source, Field: field.Name, Details: "multiple providers satisfy " + field.Type.String() + ": " + strings.Join(names, ", ")})
			}
		}
	}
	sort.Slice(g.Edges, func(i, j int) bool {
		a, b := g.Edges[i], g.Edges[j]
		if a.From != b.From { return a.From < b.From }
		if a.To != b.To { return a.To < b.To }
		return a.Field < b.Field
	})
	sort.Slice(g.Findings, func(i, j int) bool {
		a, b := g.Findings[i], g.Findings[j]
		if a.Kind != b.Kind { return a.Kind < b.Kind }
		if a.Provider != b.Provider { return a.Provider < b.Provider }
		if a.Field != b.Field { return a.Field < b.Field }
		return a.Details < b.Details
	})
	g.Cycles = cycles(g)
	return g
}

func cycles(g Graph) [][]string {
	adj := map[string][]string{}
	for _, n := range g.Nodes { adj[n.ID] = []string{} }
	for _, e := range g.Edges { adj[e.From] = append(adj[e.From], e.To) }
	for _, a := range adj { sort.Strings(a) }
	index, next := map[string]int{}, 0
	low, onStack := map[string]int{}, map[string]bool{}
	stack := []string{}
	result := [][]string{}
	var visit func(string)
	visit = func(v string) {
		index[v], low[v] = next, next
		next++
		stack = append(stack, v)
		onStack[v] = true
		for _, w := range adj[v] {
			if _, ok := index[w]; !ok {
				visit(w)
				if low[w] < low[v] { low[v] = low[w] }
			} else if onStack[w] && index[w] < low[v] { low[v] = index[w] }
		}
		if low[v] == index[v] {
			part := []string{}
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				part = append(part, w)
				if w == v { break }
			}
			sort.Strings(part)
			selfCycle := false
			if len(part) == 1 { for _, to := range adj[part[0]] { if to == part[0] { selfCycle = true } } }
			if len(part) > 1 || selfCycle { result = append(result, part) }
		}
	}
	for _, n := range g.Nodes { if _, ok := index[n.ID]; !ok { visit(n.ID) } }
	sort.Slice(result, func(i,j int) bool { return strings.Join(result[i], "\x00") < strings.Join(result[j], "\x00") })
	return result
}
func (g Graph) JSON() ([]byte,error) { return json.MarshalIndent(g, "", "  ") }
func (g Graph) DOT() string {
	var b strings.Builder
	b.WriteString("digraph NestGo {\n  rankdir=LR;\n  node [shape=box];\n")
	for _, n := range g.Nodes { fmt.Fprintf(&b, "  %q;\n", n.ID) }
	for _, e := range g.Edges { fmt.Fprintf(&b, "  %q -> %q [label=%q];\n", e.From, e.To, e.Field) }
	b.WriteString("}\n")
	return b.String()
}
