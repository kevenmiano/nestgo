package runtime

import (
 "context"
 "io"
 "net"
 "net/http"
 "testing"
 "time"

 "github.com/kevenmiano/nestgo/httpadapter"
 fastadapter "github.com/kevenmiano/nestgo/platform/fasthttp"
 "github.com/kevenmiano/nestgo/platform/nethttp"
)

func TestHTTPAdapterConformance(t *testing.T){
 factories:=map[string]func()httpadapter.Adapter{
  "fasthttp":func()httpadapter.Adapter{return fastadapter.New()},
  "nethttp":func()httpadapter.Adapter{return nethttp.New()},
 }
 for name,newAdapter:=range factories {
  t.Run(name,func(t *testing.T){
   a:=newAdapter()
   if err:=a.Register(httpadapter.Route{Method:"GET",Path:"/hello/{name}",Handler:func(ctx httpadapter.Context)error{
    ctx.Status(201)
    ctx.SetHeader("X-Engine-Parity","yes")
    return ctx.Write([]byte(ctx.Param("name")+":"+ctx.Query("q")))
   }});err!=nil{t.Fatal(err)}
   listener,err:=net.Listen("tcp","127.0.0.1:0")
   if err!=nil{t.Fatal(err)}
   done:=make(chan error,1)
   go func(){done<-a.Serve(listener)}()
   client:=&http.Client{Timeout:3*time.Second}
   req,err:=http.NewRequest(http.MethodGet,"http://"+listener.Addr().String()+"/hello/ada?q=ok",nil)
   if err!=nil{t.Fatal(err)}
   resp,err:=client.Do(req)
   if err!=nil{t.Fatal(err)}
   body,err:=io.ReadAll(resp.Body)
   resp.Body.Close()
   if err!=nil{t.Fatal(err)}
   if resp.StatusCode!=201||string(body)!="ada:ok"||resp.Header.Get("X-Engine-Parity")!="yes"{
    t.Fatalf("unexpected HTTP result: status=%d body=%q header=%q",resp.StatusCode,body,resp.Header.Get("X-Engine-Parity"))
   }
   ctx,cancel:=context.WithTimeout(context.Background(),3*time.Second)
   defer cancel()
   if err:=a.Shutdown(ctx);err!=nil{t.Fatal(err)}
   select{
   case <-done:
   case <-time.After(3*time.Second):t.Fatal("server failed to shutdown")
   }
  })
 }
}
