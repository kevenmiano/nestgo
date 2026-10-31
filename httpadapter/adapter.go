// Package httpadapter defines transport-independent HTTP contracts.
package httpadapter

import (
 "context"
 "net"
)

type Handler func(Context) error

type Context interface {
 Method() string
 Path() string
 Param(string) string
 Query(string) string
 Header(string) string
 Body() []byte
 Status(int)
 SetHeader(string,string)
 Write([]byte) error
}

type Route struct {
 Method string
 Path string
 Handler Handler
}

type Adapter interface {
 Register(Route) error
 Serve(net.Listener) error
 Shutdown(context.Context) error
}

// NativeInstanceProvider is optional. Applications may inspect the native server.
type NativeInstanceProvider interface { Instance() any }
