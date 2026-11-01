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

type appOptionsConsumer struct { Config *config.Service }

func TestNewWithDatabaseFunctionalOptions(t *testing.T) {
	pkg := metadata.Package{Providers: []metadata.Provider{{Type: reflect.TypeOf((*appOptionsConsumer)(nil))}}}
	app, err := New(
		WithConfig(config.WithIgnoreEnvFile()),
		WithPackages(pkg),
		WithDebug(true),
	)
	if err != nil { t.Fatal(err) }
	if app.Config == nil || app.Connections == nil || app.HTTPAdapter() == nil {
		t.Fatal("application not initialized")
	}
	if err := app.Close(); err != nil { t.Fatal(err) }
}

func TestNewRejectsInvalidDatabaseOptions(t *testing.T) {
	if _, err := New(nil); err == nil { t.Fatal("nil app option accepted") }
	if _, err := New(WithProvider(nil)); err == nil { t.Fatal("nil provider accepted") }
	if _, err := New(WithLogger(nil)); err == nil { t.Fatal("nil logger accepted") }
	_, err := New(WithConfig(config.WithEnvFile("")))
	if err == nil || !strings.Contains(err.Error(), "empty env") {
		t.Fatalf("expected config validation failure: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := New(WithContext(ctx)); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context, got %v", err)
	}
}
