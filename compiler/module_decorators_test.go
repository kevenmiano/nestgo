package compiler

import (
	"bytes"
	"strings"
	"testing"
)

func TestGenerateModuleReferencesAcrossFilesAndStableOrdering(t *testing.T) {
	first, err := Parse("first.mgo", []byte(`package demo
@Module("root")
@Imports("DatabaseModule")
@Exports("Service")
type RootModule struct{}
@Injectable()
type Service struct{}
`))
	if err != nil { t.Fatal(err) }
	second, err := Parse("second.mgo", []byte(`package demo
@Module("db")
type DatabaseModule struct{}
`))
	if err != nil { t.Fatal(err) }
	a, err := Generate([]*File{first,second})
	if err != nil { t.Fatal(err) }
	b, err := Generate([]*File{second,first})
	if err != nil { t.Fatal(err) }
	if !bytes.Equal(a,b) { t.Fatalf("metadata must be invariant under file order\n%s\n%s",a,b) }
	for _, expected := range []string{`Name: "root"`,`Imports: []reflect.Type{reflect.TypeOf((*DatabaseModule)(nil))}`,`Exports: []reflect.Type{reflect.TypeOf((*Service)(nil))}`} {
		if !strings.Contains(string(a),expected) {t.Errorf("missing %s in generated code:\n%s",expected,a)}
	}
}

func TestModuleDecoratorsDiagnostics(t *testing.T) {
	cases := []struct{ name, source, errorText string }{
		{"missing-module",`package demo
@Imports("X")
type A struct{}`,"require @Module"},
		{"missing-import",`package demo
@Module("a")
@Imports("Unknown")
type A struct{}`,"unknown local module"},
		{"missing-export",`package demo
@Module("a")
@Exports("Unknown")
type A struct{}`,"unknown local provider"},
		{"invalid-literal",`package demo
@Module("a")
@Imports(123)
type A struct{}`,"expects a quoted"},
		{"dynamic-selector",`package demo
@Module("a")
@Imports("external.Module")
type A struct{}`,"unknown local module"},
		{"duplicate-import",`package demo
@Module("a")
@Imports("B,B")
type A struct{}
@Module("b")
type B struct{}`,"duplicate module"},
		{"duplicate-annotation",`package demo
@Module("a")
@Exports("B")
@Exports("B")
type A struct{}
@Injectable()
type B struct{}`,"only one @Imports and @Exports"},
	}
	for _, tc := range cases {
		t.Run(tc.name,func(t *testing.T){
			f,err:=Parse("bad.mgo",[]byte(tc.source))
			if err==nil {_,err=Generate([]*File{f})}
			if err==nil||!strings.Contains(err.Error(),tc.errorText){t.Fatalf("wanted %q, got %v",tc.errorText,err)}
		})
	}
}

func TestExistingModuleDecoratorIsCompatible(t *testing.T) {
	file,err:=Parse("legacy.mgo",[]byte(`package demo
@Module("legacy")
type LegacyModule struct{}`))
	if err!=nil{t.Fatal(err)}
	src,err:=Generate([]*File{file})
	if err!=nil{t.Fatal(err)}
	if !strings.Contains(string(src),`Name: "legacy"`) {t.Fatal("existing module name missing")}
}
