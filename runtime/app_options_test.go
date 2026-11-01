package runtime

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/config"
	"github.com/kevenmiano/nestgo/metadata"
)

type appOptionsConsumer struct {
	Config *config.Service
}

func TestNewAppFunctionalOptions(t *testing.T) {
	pkg := metadata.Package{Providers: []metadata.Provider{{Type: reflect.TypeOf((*appOptionsConsumer)(nil))}}}
	app, err := NewApp(context.Background(),
		WithConfig(config.WithIgnoreEnvFile()),
		WithPackages(pkg),
		WithDebug(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if app.Config == nil || app.Connections == nil || app.HTTPHandler() == nil {
		t.Fatal("application not initialized")
	}
	if err := app.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestNewAppRejectsInvalidOptions(t *testing.T) {
	if _, err := NewApp(context.Background(), nil); err == nil {
		t.Fatal("nil app option accepted")
	}
	if _, err := NewApp(context.Background(), WithProvider(nil)); err == nil {
		t.Fatal("nil provider accepted")
	}
	if _, err := NewApp(context.Background(), WithLogger(nil)); err == nil {
		t.Fatal("nil logger accepted")
	}
	_, err := NewApp(context.Background(), WithConfig(config.WithEnvFile("")))
	if err == nil || !strings.Contains(err.Error(), "empty env") {
		t.Fatalf("expected config validation failure: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewApp(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}
