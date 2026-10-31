// Package logging provides opt-in Go-native logging. No configuration uses log.Default().
package logging

import (
 "context"
 "fmt"
 "io"
 "log"
 "log/slog"
 "strings"
 "sync"
 "time"
)

type Level int
const (
 LevelDebug Level = iota
 LevelInfo
 LevelWarn
 LevelError
)
func (l Level) String() string { switch l {case LevelDebug:return "DEBUG";case LevelInfo:return "LOG";case LevelWarn:return "WARN";case LevelError:return "ERROR";default:return "LOG"} }
func (l Level) slogLevel() slog.Level {switch l {case LevelDebug:return slog.LevelDebug;case LevelWarn:return slog.LevelWarn;case LevelError:return slog.LevelError;default:return slog.LevelInfo}}

// Logger is the adapter interface; implement this for any third-party backend.
type Logger interface { Log(ctx context.Context, level Level, component, message string, attrs ...slog.Attr) }

// Std delegates to the Go standard library logger without mutating global settings.
type Std struct { Logger *log.Logger }
func (s Std) Log(_ context.Context, level Level, component, message string, attrs ...slog.Attr) {
 l:=s.Logger; if l==nil {l=log.Default()}
 var b strings.Builder
 fmt.Fprintf(&b,"[%s] [%s] %s",level.String(),component,message)
 for _,a:=range attrs {fmt.Fprintf(&b," %s=%v",a.Key,a.Value.Any())}
 l.Print(b.String())
}

type Slog struct { Logger *slog.Logger }
func (s Slog) Log(ctx context.Context, level Level, component, message string, attrs ...slog.Attr) {
 l:=s.Logger;if l==nil {l=slog.Default()}
 values:=make([]slog.Attr,0,len(attrs)+1)
 values=append(values,slog.String("component",component));values=append(values,attrs...)
 if ctx==nil {ctx=context.Background()}
 l.LogAttrs(ctx,level.slogLevel(),message,values...)
}

// Pretty serializes console writes and enables ANSI color only on explicit request.
type Pretty struct {mu sync.Mutex; output io.Writer; color bool; clock func()time.Time}
func NewPretty(output io.Writer, color bool) *Pretty {if output==nil {output=io.Discard};return &Pretty{output:output,color:color,clock:time.Now}}
func (p *Pretty) Log(_ context.Context, level Level, component,message string, attrs ...slog.Attr) {
 p.mu.Lock();defer p.mu.Unlock()
 timePart:=p.clock().Format("15:04:05")
 levelPart:=level.String()
 if p.color { c:="\x1b[36m";if level==LevelWarn {c="\x1b[33m"};if level==LevelError {c="\x1b[31m"};if level==LevelDebug {c="\x1b[90m"};levelPart=c+levelPart+"\x1b[0m" }
 fmt.Fprintf(p.output,"%s %-5s [%s] %s",timePart,levelPart,component,message)
 for _,a:=range attrs {fmt.Fprintf(p.output," %s=%v",a.Key,a.Value.Any())}
 fmt.Fprintln(p.output)
}

type Config struct {Logger Logger; MinLevel Level}
type Service struct {logger Logger; min Level}
func New(cfg Config) *Service {l:=cfg.Logger;if l==nil {l=Std{}};return &Service{logger:l,min:cfg.MinLevel}}
func (s *Service) Log(ctx context.Context,level Level,component,message string,attrs ...slog.Attr){if level<s.min{return};s.logger.Log(ctx,level,component,message,attrs...)}
func (s *Service) Debug(ctx context.Context,component,message string,attrs ...slog.Attr){s.Log(ctx,LevelDebug,component,message,attrs...)}
func (s *Service) Info(ctx context.Context,component,message string,attrs ...slog.Attr){s.Log(ctx,LevelInfo,component,message,attrs...)}
func (s *Service) Warn(ctx context.Context,component,message string,attrs ...slog.Attr){s.Log(ctx,LevelWarn,component,message,attrs...)}
func (s *Service) Error(ctx context.Context,component,message string,attrs ...slog.Attr){s.Log(ctx,LevelError,component,message,attrs...)}

// Startup logs the real elapsed duration only after a bootstrap stage finishes.
func (s *Service) Startup(ctx context.Context,component,message string,started time.Time){s.Info(ctx,component,message,slog.Duration("duration",time.Since(started)))}
