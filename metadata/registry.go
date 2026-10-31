package metadata

import ("fmt";"reflect";"sync")
type Parameter struct {Index int;Kind string;Name string}
type Route struct {Method,Path,Handler string;Status int;Parameters []Parameter; StaticInvoke func(any)(any,error)}
type Controller struct {Type reflect.Type;Prefix string;Routes []Route}
type Provider struct {Type reflect.Type;Scope string}
type Module struct {Type reflect.Type;Name string}
type Package struct {Controllers []Controller;Providers []Provider;Modules []Module}
var mu sync.RWMutex
var packages=map[string]Package{}
func Register(name string,p Package){mu.Lock();defer mu.Unlock();if _,ok:=packages[name];ok{panic(fmt.Sprintf("nestgo: duplicate metadata package %s",name))};packages[name]=p}
func Lookup(name string)(Package,bool){mu.RLock();defer mu.RUnlock();p,ok:=packages[name];return p,ok}
func All()map[string]Package{mu.RLock();defer mu.RUnlock();result:=make(map[string]Package,len(packages));for k,v:=range packages{result[k]=v};return result}
