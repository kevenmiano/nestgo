package diagnostics

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/kevenmiano/nestgo/metadata"
)

// ModuleEdge models a directed import from one module to another.
type ModuleEdge struct {
	From string `json:"from"`
	To string `json:"to"`
}

// ModuleInfo is a deterministic projection of the optional metadata.Module contract.
type ModuleInfo struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Exports []string `json:"exports"`
}

// ModuleFinding records invalid, missing, or duplicate module references.
type ModuleFinding struct {
	Kind string `json:"kind"`
	Module string `json:"module"`
	Details string `json:"details"`
}

// ModuleGraph is deliberately separate from the provider injection graph.
// Schema versioning permits downstream tooling to distinguish these contracts.
type ModuleGraph struct {
	SchemaVersion int `json:"schemaVersion"`
	Modules []ModuleInfo `json:"modules"`
	Imports []ModuleEdge `json:"imports"`
	Cycles [][]string `json:"cycles"`
	Findings []ModuleFinding `json:"findings"`
}

func moduleTypeID(t reflect.Type) string {
	if t == nil { return "" }
	for t.Kind() == reflect.Ptr { t = t.Elem() }
	if t.PkgPath() == "" { return t.String() }
	if t.Name() == "" { return t.String() }
	return t.PkgPath() + "." + t.Name()
}

func AnalyzeModules(packages ...metadata.Package) ModuleGraph {
	g := ModuleGraph{SchemaVersion: 1, Modules: []ModuleInfo{}, Imports: []ModuleEdge{}, Cycles: [][]string{}, Findings: []ModuleFinding{}}
	declared := map[string]metadata.Module{}
	providers := map[string]bool{}
	for _, p := range packages {
		for _, provider := range p.Providers {
			if provider.Type != nil { providers[moduleTypeID(provider.Type)] = true }
		}
		for _, module := range p.Modules {
			id := moduleTypeID(module.Type)
			if module.Type == nil || id == "" || module.Type.Kind() != reflect.Ptr || module.Type.Elem().Kind() != reflect.Struct {
				g.Findings = append(g.Findings, ModuleFinding{Kind:"invalid-module", Module:id, Details:"module type must be pointer to struct"})
				continue
			}
			if _, exists := declared[id]; exists {
				g.Findings = append(g.Findings, ModuleFinding{Kind:"duplicate-module", Module:id, Details:"module is declared multiple times"})
				continue
			}
			declared[id] = module
		}
	}
	ids := make([]string,0,len(declared))
	for id := range declared { ids = append(ids,id) }
	sort.Strings(ids)
	seenEdges := map[ModuleEdge]bool{}
	for _, id := range ids {
		mod := declared[id]
		info := ModuleInfo{ID:id, Name:mod.Name, Exports:[]string{}}
		for _, typ := range mod.Exports {
			exported := moduleTypeID(typ)
			if exported == "" || !providers[exported] {
				g.Findings = append(g.Findings, ModuleFinding{Kind:"unknown-export",Module:id,Details:"export not registered as provider: "+exported})
				continue
			}
			info.Exports = append(info.Exports, exported)
		}
		sort.Strings(info.Exports)
		info.Exports = uniqueStrings(info.Exports)
		g.Modules = append(g.Modules, info)
		for _, imported := range mod.Imports {
			target := moduleTypeID(imported)
			if imported == nil || imported.Kind() != reflect.Ptr || imported.Elem().Kind() != reflect.Struct {
				g.Findings = append(g.Findings, ModuleFinding{Kind:"invalid-import",Module:id,Details:"import must be pointer-to-struct module type: "+target})
				continue
			}
			if _, ok := declared[target]; !ok {
				g.Findings = append(g.Findings, ModuleFinding{Kind:"missing-module",Module:id,Details:"unregistered imported module: "+target})
				continue
			}
			edge := ModuleEdge{From:id,To:target}
			if !seenEdges[edge] { seenEdges[edge]=true;g.Imports=append(g.Imports,edge) }
		}
	}
	sort.Slice(g.Imports,func(i,j int)bool{
		if g.Imports[i].From != g.Imports[j].From {return g.Imports[i].From<g.Imports[j].From}
		return g.Imports[i].To<g.Imports[j].To
	})
	sort.Slice(g.Findings,func(i,j int)bool{
		a,b:=g.Findings[i],g.Findings[j]
		if a.Kind!=b.Kind{return a.Kind<b.Kind}
		if a.Module!=b.Module{return a.Module<b.Module}
		return a.Details<b.Details
	})
	g.Cycles=moduleCycles(g)
	return g
}

func uniqueStrings(items []string) []string {
	if len(items)==0{return items}
	dest:=items[:1]
	for _,item:=range items[1:] {
		if item!=dest[len(dest)-1] {dest=append(dest,item)}
	}
	return dest
}

// moduleCycles computes deterministic strongly connected components in O(V+E).
func moduleCycles(g ModuleGraph) [][]string {
	adj:=map[string][]string{}
	for _,m:=range g.Modules {adj[m.ID]=[]string{}}
	for _,e:=range g.Imports {adj[e.From]=append(adj[e.From],e.To)}
	for _,targets:=range adj {sort.Strings(targets)}
	index:=map[string]int{}
	low:=map[string]int{}
	onStack:=map[string]bool{}
	stack:=[]string{}
	next:=0
	cycles:=[][]string{}
	var visit func(string)
	visit=func(v string){
		index[v]=next;low[v]=next;next++
		stack=append(stack,v);onStack[v]=true
		for _,w:=range adj[v] {
			if _,seen:=index[w];!seen {
				visit(w)
				if low[w]<low[v]{low[v]=low[w]}
			}else if onStack[w]&&index[w]<low[v]{low[v]=index[w]}
		}
		if low[v]!=index[v]{return}
		group:=[]string{}
		for {
			w:=stack[len(stack)-1];stack=stack[:len(stack)-1]
			onStack[w]=false
			group=append(group,w)
			if w==v {break}
		}
		sort.Strings(group)
		self:=false
		if len(group)==1 {
			for _,w:=range adj[group[0]] {if w==group[0] {self=true}}
		}
		if len(group)>1||self{cycles=append(cycles,group)}
	}
	for _,m:=range g.Modules {if _,found:=index[m.ID];!found{visit(m.ID)}}
	sort.Slice(cycles,func(i,j int)bool{return strings.Join(cycles[i],"\x00")<strings.Join(cycles[j],"\x00")})
	return cycles
}

func (g ModuleGraph) JSON() ([]byte,error) {return json.MarshalIndent(g,"","  ")}
func (g ModuleGraph) DOT() string {
	var b strings.Builder
	b.WriteString("digraph NestGoModules {\n  rankdir=LR;\n  node [shape=box];\n")
	for _,m:=range g.Modules {fmt.Fprintf(&b,"  %q [label=%q];\n",m.ID,m.Name)}
	for _,e:=range g.Imports {fmt.Fprintf(&b,"  %q -> %q;\n",e.From,e.To)}
	b.WriteString("}\n")
	return b.String()
}
