package runtime

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/metadata"
	"github.com/kevenmiano/nestgo/platform/nethttp"
)

var lifecycleEvents []string

type lifecycleHookProvider struct{}
type lifecycleHookController struct{ Provider *lifecycleHookProvider }

func (*lifecycleHookProvider) OnModuleInit(context.Context) error {
	lifecycleEvents = append(lifecycleEvents, "provider:init")
	return nil
}
func (*lifecycleHookController) OnModuleInit(context.Context) error {
	lifecycleEvents = append(lifecycleEvents, "controller:init")
	return nil
}
func (*lifecycleHookProvider) OnApplicationBootstrap(context.Context) error {
	lifecycleEvents = append(lifecycleEvents, "provider:bootstrap")
	return nil
}
func (*lifecycleHookController) OnApplicationBootstrap(context.Context) error {
	lifecycleEvents = append(lifecycleEvents, "controller:bootstrap")
	return nil
}
func (*lifecycleHookProvider) BeforeApplicationShutdown(context.Context) error {
	lifecycleEvents = append(lifecycleEvents, "provider:before")
	return nil
}
func (*lifecycleHookController) BeforeApplicationShutdown(context.Context) error {
	lifecycleEvents = append(lifecycleEvents, "controller:before")
	return nil
}
func (*lifecycleHookProvider) OnModuleDestroy(context.Context) error {
	lifecycleEvents = append(lifecycleEvents, "provider:destroy")
	return nil
}
func (*lifecycleHookController) OnModuleDestroy(context.Context) error {
	lifecycleEvents = append(lifecycleEvents, "controller:destroy")
	return nil
}
func (*lifecycleHookProvider) OnApplicationShutdown(context.Context) error {
	lifecycleEvents = append(lifecycleEvents, "provider:shutdown")
	return nil
}
func (*lifecycleHookController) OnApplicationShutdown(context.Context) error {
	lifecycleEvents = append(lifecycleEvents, "controller:shutdown")
	return nil
}
func TestApplicationLifecycleHooks(t *testing.T) {
	lifecycleEvents = nil
	p := metadata.Package{
		Providers:   []metadata.Provider{{Type: reflect.TypeOf((*lifecycleHookProvider)(nil))}},
		Controllers: []metadata.Controller{{Type: reflect.TypeOf((*lifecycleHookController)(nil))}},
	}
	app, err := New(WithAdapter(nethttp.New()), WithPackages(p))
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := []string{"provider:init", "controller:init", "provider:bootstrap", "controller:bootstrap",
		"controller:before", "provider:before", "controller:destroy", "provider:destroy", "controller:shutdown", "provider:shutdown"}
	if !reflect.DeepEqual(lifecycleEvents, want) {
		t.Fatalf("events=%v; want=%v", lifecycleEvents, want)
	}
}

type failingLifecycleProvider struct{}

func (*failingLifecycleProvider) OnModuleInit(context.Context) error {
	return errors.New("initialization rejected")
}
func TestApplicationLifecycleInitFailure(t *testing.T) {
	_, err := New(WithPackages(metadata.Package{Providers: []metadata.Provider{{Type: reflect.TypeOf((*failingLifecycleProvider)(nil))}}}))
	if err == nil || !strings.Contains(err.Error(), "OnModuleInit") {
		t.Fatalf("expected init hook failure: %v", err)
	}
}

type failingShutdownProvider struct{}

func (*failingShutdownProvider) OnModuleDestroy(context.Context) error {
	return errors.New("close failed")
}
func TestApplicationLifecycleShutdownFailure(t *testing.T) {
	app, err := New(WithPackages(metadata.Package{Providers: []metadata.Provider{{Type: reflect.TypeOf((*failingShutdownProvider)(nil))}}}))
	if err != nil {
		t.Fatal(err)
	}
	first := app.Shutdown(context.Background())
	second := app.Shutdown(context.Background())
	if first == nil || second == nil || first.Error() != second.Error() {
		t.Fatalf("shutdown errors first=%v second=%v", first, second)
	}
}

type rollbackLifecycleProvider struct{}

var rollbackCalls int

func (*rollbackLifecycleProvider) OnModuleInit(context.Context) error {
	return errors.New("startup failed")
}
func (*rollbackLifecycleProvider) OnModuleDestroy(context.Context) error { rollbackCalls++; return nil }

func TestFailedBootstrapReleasesProviders(t *testing.T) {
	rollbackCalls = 0
	_, err := New(WithPackages(metadata.Package{Providers: []metadata.Provider{{Type: reflect.TypeOf((*rollbackLifecycleProvider)(nil))}}}))
	if err == nil || !strings.Contains(err.Error(), "startup failed") {
		t.Fatalf("expected startup error: %v", err)
	}
	if rollbackCalls != 1 {
		t.Fatalf("destroy calls=%d, want 1", rollbackCalls)
	}
}
