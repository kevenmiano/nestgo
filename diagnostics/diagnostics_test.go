package diagnostics

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/metadata"
)

type cycleA struct { B *cycleB }
type cycleB struct { A *cycleA }
type cycleSelf struct { Self *cycleSelf }
type ordinary struct{}
type missing struct { NotFound *ordinary }
type lonely struct{}
type interfaceDependency interface { ping() }
type implementOne struct{}
func (*implementOne) ping() {}
type implementTwo struct{}
func (*implementTwo) ping() {}
type consumer struct { Dependency interfaceDependency }

func pkg(types ...any) metadata.Package {
	p:=metadata.Package{}
	for _,v:=range types {p.Providers=append(p.Providers,metadata.Provider{Type:reflect.TypeOf(v),Scope:"singleton"})}
	return p
}
func TestDeterministicGraphAndCycles(t *testing.T) {
	first:=Analyze(pkg((*cycleB)(nil),(*cycleA)(nil),(*cycleSelf)(nil)))
	second:=Analyze(pkg((*cycleSelf)(nil),(*cycleA)(nil),(*cycleB)(nil)))
	j1,e1:=first.JSON(); j2,e2:=second.JSON()
	if e1!=nil||e2!=nil {t.Fatalf("json: %v / %v",e1,e2)}
	if !bytes.Equal(j1,j2) || first.DOT()!=second.DOT() {t.Fatal("graph changed with provider registration order")}
	if len(first.Cycles)!=2 {t.Fatalf("expected two cyclic components, got %v",first.Cycles)}
	if len(first.Edges)!=3 {t.Fatalf("expected three edges, got %v",first.Edges)}
	if !strings.Contains(first.DOT(),"digraph NestGo") {t.Fatal("missing DOT declaration")}
}
func TestMissingAmbiguousAndDuplicate(t *testing.T) {
	g:=Analyze(pkg((*consumer)(nil),(*implementOne)(nil),(*implementTwo)(nil),(*missing)(nil),(*lonely)(nil)),pkg((*lonely)(nil)))
	got:=map[string]bool{}
	for _,f:=range g.Findings {got[f.Kind]=true}
	for _,kind:=range []string{"ambiguous-provider","missing-provider","duplicate-provider"} {
		if !got[kind] {t.Errorf("missing %s in %v",kind,g.Findings)}
	}
}
func TestEmptyGraphHasDeterministicContainers(t *testing.T) {
	g:=Analyze()
	j,_:=g.JSON()
	for _,expected:=range []string{`"nodes": []`, `"edges": []`, `"cycles": []`} {if !bytes.Contains(j,[]byte(expected)){t.Errorf("missing %s",expected)}}
}
func TestOfflineHTMLAndStartup(t *testing.T) {
	p:=pkg((*ordinary)(nil))
	m,err:=MeasureStartup("unit-test",[]metadata.Package{p},3,10,"deadbeef")
	if err!=nil {t.Fatal(err)}
	if len(m.SamplesNS)!=3 || m.P95NS<m.MedianNS || m.AllocsPerOp<0 {t.Fatalf("invalid sample stats %+v",m)}
	r:=Report{Graph:Analyze(p),Startup:m}
	content,err:=r.HTML()
	if err!=nil {t.Fatal(err)}
	for _,want:=range []string{"<svg","unit-test","Samples","deadbeef","Startup p95"} {
		if !bytes.Contains(content,[]byte(want)) {t.Errorf("missing %q",want)}
	}
	j,err:=r.JSON()
	if err!=nil || !bytes.Contains(j,[]byte("samplesNsPerOp")) {t.Fatalf("json: %v",err)}
}
