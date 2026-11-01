package runtime

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/httpresponse"
	"github.com/kevenmiano/nestgo/metadata"
	fastadapter "github.com/kevenmiano/nestgo/platform/fasthttp"
	"github.com/kevenmiano/nestgo/platform/nethttp"
)

type responseTestController struct{}
func (*responseTestController) Typed() httpresponse.Response {
	return httpresponse.Created(map[string]string{"name":"Ada"}).WithHeader("X-Result","typed").WithCookie(httpresponse.Cookie{Name:"one",Value:"1"})
}
func (*responseTestController) Decorated(header string) string { return header }
func (*responseTestController) StaticRedirect() string { return "ignored" }
func (*responseTestController) DynamicRedirect() httpresponse.Response { return httpresponse.Redirect("/override",307) }
func (*responseTestController) Explicit(w *httpresponse.Writer) { _ = w.Status(202).Header("X-Writer","yes").Send("accepted") }
func (*responseTestController) Passthrough(w *httpresponse.Writer) string { w.Status(202).Header("X-Pass","yes"); return "from-handler" }
func (*responseTestController) Cookies() httpresponse.Response {
	return httpresponse.OK("cookie").WithCookie(httpresponse.Cookie{Name:"a",Value:"1"}).WithCookie(httpresponse.Cookie{Name:"b",Value:"2"})
}
func (*responseTestController) Failure() (string,error) { return "", httpresponse.NotFound("user missing") }
func (*responseTestController) GenericFailure() (string,error) { return "",errors.New("secret") }
func (*responseTestController) Void() {}
func (*responseTestController) MissingWrite(w *httpresponse.Writer) {}
func responseTestPackage() metadata.Package {
	return metadata.Package{Controllers:[]metadata.Controller{{
		Type:reflect.TypeOf((*responseTestController)(nil)),
		Prefix:"/response",
		Routes:[]metadata.Route{
			{Method:"GET",Path:"/typed",Handler:"Typed"},
			{Method:"GET",Path:"/decorated",Handler:"Decorated",Status:201,Headers:[]metadata.Header{{Name:"X-Decorated",Value:"yes"}},Parameters:[]metadata.Parameter{{Index:0,Kind:"Headers",Name:"X-Input"}}},
			{Method:"GET",Path:"/redirect",Handler:"StaticRedirect",RedirectURL:"/dest",RedirectStatus:301},
			{Method:"GET",Path:"/dynamic",Handler:"DynamicRedirect",RedirectURL:"/dest",RedirectStatus:301},
			{Method:"GET",Path:"/explicit",Handler:"Explicit",Parameters:[]metadata.Parameter{{Index:0,Kind:"Res"}}},
			{Method:"GET",Path:"/passthrough",Handler:"Passthrough",Parameters:[]metadata.Parameter{{Index:0,Kind:"Res",Name:"passthrough"}}},
			{Method:"GET",Path:"/cookies",Handler:"Cookies"},
			{Method:"GET",Path:"/error",Handler:"Failure"},
			{Method:"GET",Path:"/generic-error",Handler:"GenericFailure"},
			{Method:"GET",Path:"/void",Handler:"Void"},
			{Method:"GET",Path:"/missing-write",Handler:"MissingWrite",Parameters:[]metadata.Parameter{{Index:0,Kind:"Res"}}},
		},
	}}}
}
func TestHTTPResponseEngineParity(t *testing.T) {
	for _, engine := range []struct{name string; factory func()httpadapter.Adapter}{
		{"fasthttp",func()httpadapter.Adapter{return fastadapter.New()}},
		{"nethttp",func()httpadapter.Adapter{return nethttp.New()}},
	} {
		t.Run(engine.name,func(t *testing.T){
			app,err:=NewApplication(ApplicationOptions{Adapter:engine.factory()},responseTestPackage())
			if err!=nil {t.Fatal(err)}
			l,err:=net.Listen("tcp","127.0.0.1:0");if err!=nil{t.Fatal(err)}
			done:=make(chan error,1)
			go func(){done<-app.Serve(l)}()
			defer func(){
				ctx,cancel:=context.WithTimeout(context.Background(),3*time.Second);defer cancel()
				_ = app.Shutdown(ctx)
				<-done
			}()
			checkResponseCases(t,"http://"+l.Addr().String())
		})
	}
}
func TestLegacyHTTPResponseParity(t *testing.T) {
	h,err:=New(responseTestPackage());if err!=nil{t.Fatal(err)}
	// The legacy handler is used directly with a real net/http listener.
	l,err:=net.Listen("tcp","127.0.0.1:0");if err!=nil{t.Fatal(err)}
	s:=&http.Server{Handler:h}
	done:=make(chan error,1)
	go func(){done<-s.Serve(l)}()
	defer func(){
		ctx,cancel:=context.WithTimeout(context.Background(),3*time.Second);defer cancel()
		_ = s.Shutdown(ctx)
		<-done
	}()
	checkResponseCases(t,"http://"+l.Addr().String())
}
func checkResponseCases(t *testing.T, base string) {
	t.Helper()
	client:=&http.Client{Timeout:3*time.Second,CheckRedirect:func(*http.Request,[]*http.Request)error{return http.ErrUseLastResponse}}
	for _,tt:=range []struct{name,path,input string;status int;body,location,header,headerValue string;cookieCount int}{
		{"typed","/typed","",201,`{"name":"Ada"}`+"\n","","X-Result","typed",1},
		{"decorated","/decorated","test",201,`"test"`+"\n","","X-Decorated","yes",0},
		{"redirect","/redirect","",301,"","/dest","","",0},
		{"dynamic","/dynamic","",307,"","/override","","",0},
		{"explicit","/explicit","",202,"accepted","","X-Writer","yes",0},
		{"passthrough","/passthrough","",202,`"from-handler"`+"\n","","X-Pass","yes",0},
		{"cookies","/cookies","",200,`"cookie"`+"\n","","","",2},
		{"error","/error","",404,`{"statusCode":404,"message":"user missing"}`+"\n","","","",0},
		{"generic","/generic-error","",500,`{"statusCode":500,"message":"Internal server error"}`+"\n","","","",0},
		{"void","/void","",200,"null\n","","","",0},
		{"missing-write","/missing-write","",500,`{"statusCode":500,"message":"explicit @Res() handler did not write a response"}`+"\n","","","",0},
	} {
		t.Run(tt.name,func(t *testing.T){
			req,err:=http.NewRequest("GET",base+"/response"+tt.path,nil);if err!=nil{t.Fatal(err)}
			if tt.input!=""{req.Header.Set("X-Input",tt.input)}
			res,err:=client.Do(req);if err!=nil{t.Fatal(err)}
			body,err:=io.ReadAll(res.Body);res.Body.Close();if err!=nil{t.Fatal(err)}
			if res.StatusCode!=tt.status||string(body)!=tt.body {
				t.Fatalf("status %d body %q; want %d %q",res.StatusCode,string(body),tt.status,tt.body)
			}
			if tt.location!=""&&res.Header.Get("Location")!=tt.location{t.Fatalf("Location %q",res.Header.Get("Location"))}
			if tt.header!=""&&res.Header.Get(tt.header)!=tt.headerValue{t.Fatalf("%s: %q",tt.header,res.Header.Get(tt.header))}
			if tt.cookieCount!=len(res.Header.Values("Set-Cookie")){t.Fatalf("cookies: %v",res.Header.Values("Set-Cookie"))}
			if tt.name=="generic"&&strings.Contains(string(body),"secret"){t.Fatal("internal error leaked")}
		})
	}
}
func TestHTTPResponseInvalidResArgument(t *testing.T) {
	type wrong struct{}
	_ = wrong{}
	a:=&capturedAdapter{}
	p:=responseTestPackage()
	p.Controllers[0].Routes[0].Parameters=[]metadata.Parameter{{Index:0,Kind:"Res"}}
	if err:=registerRoutes(a,[]metadata.Package{p});err==nil{t.Fatal("expected bootstrap signature validation")}
}
