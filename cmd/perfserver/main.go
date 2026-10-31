package main

import (
 "flag"
 "log"
 "net/http"
 "encoding/json"

 _ "github.com/kevenmiano/nestgo/examples/shop"
 "github.com/kevenmiano/nestgo/runtime"
)

func main() {
 addr:=flag.String("addr","127.0.0.1:18080","HTTP listen address")
 mode:=flag.String("mode","nestgo","nestgo or nethttp")
 flag.Parse()
 var handler http.Handler
 if *mode=="nestgo" {
  h,err:=runtime.FromRegistry("shop")
  if err!=nil {log.Fatal(err)}
  handler=h
 } else if *mode=="nethttp" {
  mux:=http.NewServeMux()
  mux.HandleFunc("GET /users/",func(w http.ResponseWriter,r *http.Request){
   w.Header().Set("Content-Type","application/json")
   _=json.NewEncoder(w).Encode([]string{"Ada","Grace"})
  })
  handler=mux
 } else {log.Fatalf("unknown mode %q",*mode)}
 log.Printf("mode=%s addr=%s",*mode,*addr)
 log.Fatal(http.ListenAndServe(*addr,handler))
}
