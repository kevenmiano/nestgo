package runtime

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/kevenmiano/nestgo/metadata"
)

type p0BootstrapContextProvider struct{}

func (*p0BootstrapContextProvider) OnModuleInit(ctx context.Context) error {
	return ctx.Err()
}

type p0CloseLifecycleProvider struct{}

var p0CloseCalls atomic.Int64

func (*p0CloseLifecycleProvider) OnModuleDestroy(context.Context) error {
	p0CloseCalls.Add(1)
	return nil
}

func TestP0BootstrapContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := New(WithContext(ctx), WithPackages(metadata.Package{
		Providers: []metadata.Provider{{Type: reflect.TypeOf((*p0BootstrapContextProvider)(nil))}},
	}))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("bootstrap should preserve context cancellation: %v", err)
	}
}

func TestP0RejectNilBootstrapContext(t *testing.T) {
	if _, err := New(WithContext(nil)); err == nil || !strings.Contains(err.Error(), "nil bootstrap context") {
		t.Fatalf("expected nil context validation: %v", err)
	}
}

func TestP0ApplicationCloseIsIdempotent(t *testing.T) {
	p0CloseCalls.Store(0)
	app, err := New(WithPackages(metadata.Package{
		Providers: []metadata.Provider{{Type: reflect.TypeOf((*p0CloseLifecycleProvider)(nil))}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Close(); err != nil {
		t.Fatal(err)
	}
	if err := app.Close(); err != nil {
		t.Fatal(err)
	}
	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if count := p0CloseCalls.Load(); count != 1 {
		t.Fatalf("destroy hook called %d times, expected once", count)
	}
	var nilApp *Application
	if err := nilApp.Close(); err == nil {
		t.Fatal("nil application close must report initialization failure")
	}
}
