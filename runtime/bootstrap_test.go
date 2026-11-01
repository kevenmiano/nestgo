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
	Config *config.Service
	Connections *database.Connections
}

func TestCanonicalNewInjectsConfigAndConnections(t *testing.T) {
	app, err := New(
		WithConfig(config.WithIgnoreEnvFile()),
		WithPackages(metadata.Package{Providers: []metadata.Provider{{Type: reflect.TypeOf((*bootstrapConsumer)(nil))}}}),
	)
	if err != nil { t.Fatal(err) }
	defer app.Close()
	if app.Config == nil || app.Connections == nil || app.HTTPAdapter() == nil {
		t.Fatal("New did not initialize runtime services")
	}
}

func TestCanonicalNewRejectsInvalidSQLBootstrap(t *testing.T) {
	_, err := New(WithConfig(nil))
	if err == nil || !strings.Contains(err.Error(), "nil option") {
		t.Fatalf("expected config validation error, got %v", err)
	}
	_, err = New(
		WithConfig(config.WithIgnoreEnvFile()),
		WithDatabase(database.WithConnection("bad", database.WithDriver("invalid"), database.WithDSN("hidden-secret"))),
	)
	if err == nil { t.Fatal("expected invalid driver bootstrap failure") }
	if strings.Contains(err.Error(), "hidden-secret") { t.Fatal("database bootstrap leaked credentials") }
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = New(WithContext(ctx))
	if !errors.Is(err, context.Canceled) { t.Fatalf("expected cancellation, got %v", err) }
}

func TestCanonicalNewRejectsDuplicateProviders(t *testing.T) {
	pkg := metadata.Package{Providers: []metadata.Provider{
		{Type: reflect.TypeOf((*bootstrapConsumer)(nil))},
		{Type: reflect.TypeOf((*bootstrapConsumer)(nil))},
	}}
	_, err := New(WithConfig(config.WithIgnoreEnvFile()), WithPackages(pkg))
	if err == nil || !strings.Contains(err.Error(), "duplicate provider") {
		t.Fatalf("expected DI failure, got %v", err)
	}
}
