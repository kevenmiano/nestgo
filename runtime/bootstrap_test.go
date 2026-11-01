package runtime

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/config"
	"github.com/kevenmiano/nestgo/database"
	"github.com/kevenmiano/nestgo/metadata"
)

type bootstrapConsumer struct {
	Config      *config.Service
	Connections *database.Connections
}

func TestBootstrapWiresOwnedProviders(t *testing.T) {
	app, err := Bootstrap(context.Background(), BootstrapOptions{
		ConfigOptions: []config.Option{config.WithIgnoreEnvFile()},
	}, metadata.Package{
		Providers: []metadata.Provider{{Type: reflect.TypeOf((*bootstrapConsumer)(nil))}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	if app.Config == nil || app.Connections == nil || app.Handler == nil {
		t.Fatal("bootstrap did not initialize runtime services")
	}
}

func TestBootstrapRejectsInvalidOptions(t *testing.T) {
	_, err := Bootstrap(context.Background(), BootstrapOptions{
		ConfigOptions: []config.Option{nil},
	})
	if err == nil || !strings.Contains(err.Error(), "nil option") {
		t.Fatalf("expected config validation error, got %v", err)
	}
	_, err = Bootstrap(context.Background(), BootstrapOptions{
		ConfigOptions: []config.Option{config.WithIgnoreEnvFile()},
		DatabaseOptions: []database.Option{database.WithConnection("bad", database.WithDriver("invalid"), database.WithDSN("hidden-secret"))},
	})
	if err == nil {
		t.Fatal("expected invalid driver bootstrap failure")
	}
	if strings.Contains(err.Error(), "hidden-secret") {
		t.Fatal("database bootstrap leaked credentials")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = Bootstrap(ctx, BootstrapOptions{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context, got %v", err)
	}
}

func TestBootstrapClosesConnectionsWhenDIInvalid(t *testing.T) {
	_, err := Bootstrap(context.Background(), BootstrapOptions{
		ConfigOptions: []config.Option{config.WithIgnoreEnvFile()},
	}, metadata.Package{Providers: []metadata.Provider{{Type: reflect.TypeOf((*bootstrapConsumer)(nil))}, {Type: reflect.TypeOf((*bootstrapConsumer)(nil))}}})
	if err == nil || !strings.Contains(err.Error(), "duplicate provider") {
		t.Fatalf("expected DI failure, got %v", err)
	}
}
