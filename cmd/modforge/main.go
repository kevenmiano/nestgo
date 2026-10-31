package main

import (
 "github.com/kevenmiano/nestgo/compiler"
 "fmt"
 "os"
 "path/filepath"
 "strings"
)
func main() {
 if len(os.Args)<3 || os.Args[1]!="compile" {fmt.Fprintln(os.Stderr,"usage: modforge compile <directory|file.mgo>");os.Exit(2)}
 path:=os.Args[2]
 info,err:=os.Stat(path);if err!=nil {fmt.Fprintln(os.Stderr,err);os.Exit(1)}
 if !info.IsDir() {if !strings.HasSuffix(path,".mgo"){fmt.Fprintln(os.Stderr,"expected .mgo file");os.Exit(2)};path=filepath.Dir(path)}
 if err:=compiler.CompileDir(path);err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}
 fmt.Println("compiled",path)
}
