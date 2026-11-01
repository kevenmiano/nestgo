package runtime

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/kevenmiano/nestgo/metadata"
	"github.com/kevenmiano/nestgo/platform/fasthttp"
	"github.com/kevenmiano/nestgo/platform/nethttp"
)

var dependencyHookEvents []string

type lifecycleZetaStorage struct{}
type lifecycleAlphaService struct {
	Storage *lifecycleZetaStorage `nestgo:"inject"`
}
type lifecycleBetaController struct {
	Service *lifecycleAlphaService `nestgo:"inject"`
}

func (*lifecycleZetaStorage) OnModuleInit(context.Context) error {
	dependencyHookEvents = append(dependencyHookEvents, "storage:init")
	return nil
}
func (*lifecycleAlphaService) OnModuleInit(context.Context) error {
	dependencyHookEvents = append(dependencyHookEvents, "service:init")
	return nil
}
func (*lifecycleBetaController) OnModuleInit(context.Context) error {
	dependencyHookEvents = append(dependencyHookEvents, "controller:init")
	return nil
}
func (*lifecycleZetaStorage) OnModuleDestroy(context.Context) error {
	dependencyHookEvents = append(dependencyHookEvents, "storage:destroy")
	return nil
}
func (*lifecycleAlphaService) OnModuleDestroy(context.Context) error {
	dependencyHookEvents = append(dependencyHookEvents, "service:destroy")
	return nil
}
func (*lifecycleBetaController) OnModuleDestroy(context.Context) error {
	dependencyHookEvents = append(dependencyHookEvents, "controller:destroy")
	return nil
}
func TestLifecycleTopologicalProviderOrderAcrossAdapters(t *testing.T) {
	for _, adapter := range []struct {
		name string
		make func() interface{ Shutdown(context.Context) error }
	}{
		{"fasthttp", func() interface{ Shutdown(context.Context) error } { return fasthttp.New() }},
		{"nethttp", func() interface{ Shutdown(context.Context) error } { return nethttp.New() }},
	} {
		t.Run(adapter.name, func(t *testing.T) {
			dependencyHookEvents = nil
			// Alphabetical order would place AlphaService before ZetaStorage.
			pkg := metadata.Package{
				Providers: []metadata.Provider{
					{Type: reflect.TypeOf((*lifecycleAlphaService)(nil))},
					{Type: reflect.TypeOf((*lifecycleZetaStorage)(nil))},
				},
				Controllers: []metadata.Controller{{Type: reflect.TypeOf((*lifecycleBetaController)(nil))}},
			}
			var app *Application
			var err error
			switch adapter.name {
			case "fasthttp":
				app, err = New(WithPackages(pkg), WithAdapter(fasthttp.New()))
			default:
				app, err = New(WithPackages(pkg), WithAdapter(nethttp.New()))
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := app.Shutdown(context.Background()); err != nil {
				t.Fatal(err)
			}
			want := []string{"storage:init", "service:init", "controller:init", "controller:destroy", "service:destroy", "storage:destroy"}
			if !reflect.DeepEqual(dependencyHookEvents, want) {
				t.Fatalf("hook order=%v, want %v", dependencyHookEvents, want)
			}
		})
	}
}

type lifecycleStoreInterface interface{ StorageName() string }
type lifecycleInterfaceZetaStorage struct{}

func (*lifecycleInterfaceZetaStorage) StorageName() string { return "store" }
func (*lifecycleInterfaceZetaStorage) OnModuleInit(context.Context) error {
	dependencyHookEvents = append(dependencyHookEvents, "interface:store")
	return nil
}

type lifecycleInterfaceAlphaService struct {
	Store    lifecycleStoreInterface        `nestgo:"inject"`
	Optional *lifecycleUnregisteredProvider `nestgo:"optional"`
}
type lifecycleUnregisteredProvider struct{}

func (*lifecycleInterfaceAlphaService) OnModuleInit(context.Context) error {
	dependencyHookEvents = append(dependencyHookEvents, "interface:consumer")
	return nil
}
func TestLifecycleResolvedInterfaceAndOptionalDependencyOrder(t *testing.T) {
	dependencyHookEvents = nil
	pkg := metadata.Package{Providers: []metadata.Provider{
		{Type: reflect.TypeOf((*lifecycleInterfaceAlphaService)(nil))},
		{Type: reflect.TypeOf((*lifecycleInterfaceZetaStorage)(nil))},
	}}
	graph, err := prepareBootstrap([]metadata.Package{pkg}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var want = reflect.TypeOf((*lifecycleInterfaceZetaStorage)(nil))
	if graph.orderedProviders[0] != want {
		t.Fatalf("first provider=%v, want %v", graph.orderedProviders[0], want)
	}
	consumer := graph.providers[reflect.TypeOf((*lifecycleInterfaceAlphaService)(nil))].Interface().(*lifecycleInterfaceAlphaService)
	if consumer.Store == nil || consumer.Optional != nil {
		t.Fatalf("binding lost: %+v", consumer)
	}
	app, err := New(WithPackages(pkg))
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(dependencyHookEvents, []string{"interface:store", "interface:consumer"}) {
		t.Fatalf("interface hooks: %v", dependencyHookEvents)
	}
}

type lifecycleModuleRoot struct{}
type lifecycleModuleImported struct{}
type lifecycleIndependentRoot struct{}
type lifecycleIndependentImported struct{}

func (*lifecycleIndependentRoot) OnModuleInit(context.Context) error {
	dependencyHookEvents = append(dependencyHookEvents, "module:root")
	return nil
}
func (*lifecycleIndependentImported) OnModuleInit(context.Context) error {
	dependencyHookEvents = append(dependencyHookEvents, "module:imported")
	return nil
}
func TestLifecycleModuleImportOrderForIndependentProviders(t *testing.T) {
	dependencyHookEvents = nil
	root, imported := reflect.TypeOf((*lifecycleModuleRoot)(nil)), reflect.TypeOf((*lifecycleModuleImported)(nil))
	rootProvider, importProvider := reflect.TypeOf((*lifecycleIndependentRoot)(nil)), reflect.TypeOf((*lifecycleIndependentImported)(nil))
	pkg := metadata.Package{
		Providers: []metadata.Provider{{Type: rootProvider}, {Type: importProvider}},
		Modules: []metadata.Module{
			{Type: root, Imports: []reflect.Type{imported}, Providers: []reflect.Type{rootProvider}},
			{Type: imported, Providers: []reflect.Type{importProvider}},
		},
	}
	app, err := New(WithPackages(pkg))
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Close(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(dependencyHookEvents, []string{"module:imported", "module:root"}) {
		t.Fatalf("imported module must initialize first: %v", dependencyHookEvents)
	}
}

type lifecycleCycleA struct {
	B *lifecycleCycleB `nestgo:"inject"`
}
type lifecycleCycleB struct {
	A *lifecycleCycleA `nestgo:"inject"`
}
type lifecycleSelfCycle struct {
	Self *lifecycleSelfCycle `nestgo:"inject"`
}

func TestLifecycleRejectsProviderCyclesBeforeHooks(t *testing.T) {
	for _, tc := range []struct {
		name      string
		providers []metadata.Provider
		want      string
	}{
		{"indirect", []metadata.Provider{{Type: reflect.TypeOf((*lifecycleCycleA)(nil))}, {Type: reflect.TypeOf((*lifecycleCycleB)(nil))}}, "lifecycleCycleA.B"},
		{"self", []metadata.Provider{{Type: reflect.TypeOf((*lifecycleSelfCycle)(nil))}}, "lifecycleSelfCycle.Self"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var first string
			for i := 0; i < 3; i++ {
				pkg := metadata.Package{Providers: tc.providers}
				_, err := New(WithPackages(pkg))
				var injection *InjectionError
				if !errors.As(err, &injection) || injection.Code != "dependency_cycle" || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("want typed dependency cycle, got %v", err)
				}
				if i == 0 {
					first = err.Error()
				} else if err.Error() != first {
					t.Fatalf("nondeterministic cycle: %q vs %q", first, err.Error())
				}
			}
		})
	}
}

type lifecycleCancelA struct {
	Cancel context.CancelFunc `nestgo:"-"`
}
type lifecycleCancelZ struct{}

var cancelNextHookCalls atomic.Int64
var cancelCleanupCalls atomic.Int64
var cancelCleanupContextErr atomic.Bool

func (*lifecycleCancelA) OnModuleInit(ctx context.Context) error {
	if fn, ok := ctx.Value(lifecycleCancelKey{}).(context.CancelFunc); ok {
		fn()
	}
	return nil
}

type lifecycleCancelKey struct{}

func (*lifecycleCancelZ) OnModuleInit(context.Context) error {
	cancelNextHookCalls.Add(1)
	return nil
}
func (*lifecycleCancelA) OnModuleDestroy(ctx context.Context) error {
	cancelCleanupCalls.Add(1)
	if ctx.Err() != nil {
		cancelCleanupContextErr.Store(true)
	}
	return nil
}
func (*lifecycleCancelZ) OnModuleDestroy(ctx context.Context) error {
	cancelCleanupCalls.Add(1)
	if ctx.Err() != nil {
		cancelCleanupContextErr.Store(true)
	}
	return nil
}
func TestLifecycleCanceledBootstrapRollsBackWithIndependentContext(t *testing.T) {
	cancelNextHookCalls.Store(0)
	cancelCleanupCalls.Store(0)
	cancelCleanupContextErr.Store(false)
	inner, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := context.WithValue(inner, lifecycleCancelKey{}, context.CancelFunc(cancel))
	pkg := metadata.Package{Providers: []metadata.Provider{{Type: reflect.TypeOf((*lifecycleCancelA)(nil))}, {Type: reflect.TypeOf((*lifecycleCancelZ)(nil))}}}
	_, err := New(WithContext(ctx), WithPackages(pkg))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want cancellation, got %v", err)
	}
	if cancelNextHookCalls.Load() != 0 {
		t.Fatal("hooks continued after cancellation")
	}
	if cancelCleanupCalls.Load() != 2 || cancelCleanupContextErr.Load() {
		t.Fatalf("rollback not completed with live context: calls=%d canceled=%t", cancelCleanupCalls.Load(), cancelCleanupContextErr.Load())
	}
}

type lifecycleConcurrentProvider struct{}

var lifecycleConcurrentShutdownCalls atomic.Int64

func (*lifecycleConcurrentProvider) OnModuleDestroy(context.Context) error {
	lifecycleConcurrentShutdownCalls.Add(1)
	return nil
}
func TestLifecycleConcurrentShutdownInvokedOnce(t *testing.T) {
	lifecycleConcurrentShutdownCalls.Store(0)
	pkg := metadata.Package{Providers: []metadata.Provider{{Type: reflect.TypeOf((*lifecycleConcurrentProvider)(nil))}}}
	app, err := New(WithPackages(pkg))
	if err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for i := 0; i < 32; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			if err := app.Shutdown(context.Background()); err != nil {
				t.Errorf("Shutdown: %v", err)
			}
		}()
	}
	group.Wait()
	if lifecycleConcurrentShutdownCalls.Load() != 1 {
		t.Fatalf("cleanup called %d times", lifecycleConcurrentShutdownCalls.Load())
	}
}

type lifecycleUnregisteredController struct {
	Missing *lifecycleUnregisteredProvider `nestgo:"inject"`
}

var lifecycleValidationHookCalls atomic.Int64

type lifecycleValidationProvider struct{}

func (*lifecycleValidationProvider) OnModuleInit(context.Context) error {
	lifecycleValidationHookCalls.Add(1)
	return nil
}
func (*lifecycleValidationProvider) OnModuleDestroy(context.Context) error {
	lifecycleValidationHookCalls.Add(10)
	return nil
}
func TestLifecycleNoInitHooksBeforeControllerValidation(t *testing.T) {
	lifecycleValidationHookCalls.Store(0)
	pkg := metadata.Package{
		Providers:   []metadata.Provider{{Type: reflect.TypeOf((*lifecycleValidationProvider)(nil))}},
		Controllers: []metadata.Controller{{Type: reflect.TypeOf((*lifecycleUnregisteredController)(nil))}},
	}
	if _, err := New(WithPackages(pkg)); err == nil {
		t.Fatal("missing controller dependency accepted")
	}
	if lifecycleValidationHookCalls.Load() != 10 {
		t.Fatalf("initialized before validation or skipped rollback: calls=%d", lifecycleValidationHookCalls.Load())
	}
}

type lifecycleZeroA struct{}
type lifecycleZeroB struct{}
type lifecycleZeroConsumer struct {
	Source *lifecycleZeroA `nestgo:"inject"`
}

func TestLifecycleZeroSizedProvidersDoNotAliasDependencyIdentity(t *testing.T) {
	a := reflect.TypeOf((*lifecycleZeroA)(nil))
	b := reflect.TypeOf((*lifecycleZeroB)(nil))
	consumer := reflect.TypeOf((*lifecycleZeroConsumer)(nil))
	pkg := metadata.Package{Providers: []metadata.Provider{{Type: consumer}, {Type: b}, {Type: a}}}
	for i := 0; i < 20; i++ {
		graph, err := prepareBootstrap([]metadata.Package{pkg}, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		var indexA, indexB, indexConsumer int
		for j, typ := range graph.orderedProviders {
			if typ == a {
				indexA = j
			}
			if typ == b {
				indexB = j
			}
			if typ == consumer {
				indexConsumer = j
			}
		}
		if indexA >= indexConsumer || indexB > indexConsumer {
			t.Fatalf("invalid dependency order %v (wanted concrete zero-size A before consumer)", graph.orderedProviders)
		}
	}
}

type lifecyclePartialFactoryResource struct{}
type lifecycleFailingFactoryResource struct{}

var lifecycleFactoryRollbackCount atomic.Int64

func (*lifecyclePartialFactoryResource) OnModuleDestroy(context.Context) error {
	lifecycleFactoryRollbackCount.Add(1)
	return nil
}

func TestLifecycleRollsBackSuccessfulFactoryOnSubsequentFailure(t *testing.T) {
	lifecycleFactoryRollbackCount.Store(0)
	pkg := metadata.Package{Providers: []metadata.Provider{
		{
			Type: reflect.TypeOf((*lifecyclePartialFactoryResource)(nil)),
			Factory: func(resolve func(reflect.Type) (any, bool)) (any, error) {
				return &lifecyclePartialFactoryResource{}, nil
			},
		},
		{
			Type: reflect.TypeOf((*lifecycleFailingFactoryResource)(nil)),
			Factory: func(resolve func(reflect.Type) (any, bool)) (any, error) {
				return nil, errors.New("simulated factory failure")
			},
		},
	}}
	_, err := New(WithPackages(pkg))
	if err == nil || !strings.Contains(err.Error(), "simulated factory failure") {
		t.Fatalf("factory error lost: %v", err)
	}
	if got := lifecycleFactoryRollbackCount.Load(); got != 1 {
		t.Fatalf("factory resource closed %d times, want 1", got)
	}
}

type lifecycleCallerOwnedService struct{}
var lifecycleCallerCleanupCount atomic.Int64

func (*lifecycleCallerOwnedService) OnModuleDestroy(context.Context) error {
	lifecycleCallerCleanupCount.Add(1)
	return nil
}

func TestLifecycleRollsBackExternalProviderOnInvalidController(t *testing.T) {
	lifecycleCallerCleanupCount.Store(0)
	pkg := metadata.Package{Controllers: []metadata.Controller{{
		Type: reflect.TypeOf((*lifecycleUnregisteredController)(nil)),
	}}}
	_, err := New(WithPackages(pkg), WithProvider(&lifecycleCallerOwnedService{}))
	if err == nil || !strings.Contains(err.Error(), "missing required dependency") {
		t.Fatalf("invalid controller accepted: %v", err)
	}
	if got := lifecycleCallerCleanupCount.Load(); got != 1 {
		t.Fatalf("external provider cleanup invoked %d times", got)
	}
}
