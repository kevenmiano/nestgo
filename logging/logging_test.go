package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestDefaultUsesStandardLog(t *testing.T) {
	var b bytes.Buffer
	old := log.Writer()
	log.SetOutput(&b)
	defer log.SetOutput(old)
	New(Config{}).Info(context.Background(), "App", "ready")
	if !strings.Contains(b.String(), "[LOG] [App] ready") {
		t.Fatal(b.String())
	}
}
func TestPrettyAndLevels(t *testing.T) {
	var b bytes.Buffer
	p := NewPretty(&b, false)
	s := New(Config{Logger: p, MinLevel: LevelInfo})
	s.Debug(context.Background(), "Boot", "hidden")
	s.Warn(context.Background(), "Boot", "up", slog.Int("port", 3000))
	if strings.Contains(b.String(), "hidden") || !strings.Contains(b.String(), "port=3000") || strings.Contains(b.String(), "\x1b[") {
		t.Fatal(b.String())
	}
}
func TestPrettyColorOptIn(t *testing.T) {
	var b bytes.Buffer
	NewPretty(&b, true).Log(context.Background(), LevelError, "App", "fail")
	if !strings.Contains(b.String(), "\x1b[31m") {
		t.Fatal(b.String())
	}
}
func TestSlogJSON(t *testing.T) {
	var b bytes.Buffer
	s := New(Config{Logger: Slog{Logger: slog.New(slog.NewJSONHandler(&b, nil))}})
	s.Info(context.Background(), "Users", "created", slog.Int("id", 7))
	var m map[string]any
	if err := json.Unmarshal(b.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m["component"] != "Users" || m["id"] != float64(7) {
		t.Fatal(m)
	}
}
func TestStartupDuration(t *testing.T) {
	var b bytes.Buffer
	s := New(Config{Logger: NewPretty(&b, false)})
	s.Startup(context.Background(), "Loader", "initialized", time.Now().Add(-time.Millisecond))
	if !strings.Contains(b.String(), "duration=") {
		t.Fatal(b.String())
	}
}
