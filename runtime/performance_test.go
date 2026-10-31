package runtime

import (
 "fmt"
 "net/http"
 "net/http/httptest"
 "reflect"
 "testing"

 "github.com/kevenmiano/nestgo/metadata"
)

type benchmarkController struct{}
func (*benchmarkController) Ping() string { return "pong" }

type benchmarkService interface { Value() string }
type benchmarkImplementation struct{}
func (*benchmarkImplementation) Value() string { return "ok" }
type benchmarkConsumer struct { Dependency benchmarkService }
type benchmarkAmbiguous struct{}
func (*benchmarkAmbiguous) Value() string { return "other" }

func TestDynamicInterfaceResolutionIsPreserved(t *testing.T) {
 pkg := metadata.Package{Providers: []metadata.Provider{
  {Type: reflect.TypeOf((*benchmarkImplementation)(nil))},
  {Type: reflect.TypeOf((*benchmarkConsumer)(nil))},
 }}
 if _,err:=New(pkg);err!=nil { t.Fatalf("expected dynamic interface injection to work: %v",err) }
 pkg.Providers=append(pkg.Providers, metadata.Provider{Type: reflect.TypeOf((*benchmarkAmbiguous)(nil))})
 if _,err:=New(pkg);err==nil { t.Fatal("ambiguous dynamic providers should still fail") }
}

func makeProviderFixture(n int) metadata.Package {
 providers:=make([]metadata.Provider,0,n)
 for i:=0;i<n;i++ {
  // Distinct structs; type generation runs outside the timed benchmark loop.
  typ:=reflect.StructOf([]reflect.StructField{{
   Name:"Value",Type:reflect.TypeOf(""),Tag:reflect.StructTag(fmt.Sprintf("json:%q",fmt.Sprintf("value_%d",i))),
  }})
  providers=append(providers,metadata.Provider{Type:reflect.PointerTo(typ)})
 }
 return metadata.Package{Providers:providers}
}

func BenchmarkStartupProviders(b *testing.B) {
 for _,n:=range []int{1,10,100,1000} {
  b.Run(fmt.Sprintf("providers_%d",n),func(b *testing.B) {
   pkg:=makeProviderFixture(n)
   b.ReportAllocs()
   b.ResetTimer()
   for i:=0;i<b.N;i++ {if _,err:=New(pkg);err!=nil{b.Fatal(err)}}
  })
 }
}

func BenchmarkHTTPRequestReflection(b *testing.B) {
 pkg:=metadata.Package{Controllers:[]metadata.Controller{{
  Type:reflect.TypeOf((*benchmarkController)(nil)),
  Prefix:"/benchmark",Routes:[]metadata.Route{{Method:"GET",Path:"/ping",Handler:"Ping"}},
 }}}
 h,err:=New(pkg);if err!=nil{b.Fatal(err)}
 req:=httptest.NewRequest(http.MethodGet,"/benchmark/ping",nil)
 b.ReportAllocs()
 b.ResetTimer()
 for i:=0;i<b.N;i++{
  response:=httptest.NewRecorder()
  h.ServeHTTP(response,req)
  if response.Code!=200{b.Fatalf("unexpected response: %d",response.Code)}
 }
}

func BenchmarkHTTPRequestNetHTTP(b *testing.B) {
 h:=http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  w.Header().Set("Content-Type","application/json")
  _,_ = w.Write([]byte{34, 112, 111, 110, 103, 34, 10})
 })
 req:=httptest.NewRequest(http.MethodGet,"/benchmark/ping",nil)
 b.ReportAllocs()
 b.ResetTimer()
 for i:=0;i<b.N;i++{
  response:=httptest.NewRecorder()
  h.ServeHTTP(response,req)
  if response.Code!=200{b.Fatalf("unexpected response: %d",response.Code)}
 }
}
