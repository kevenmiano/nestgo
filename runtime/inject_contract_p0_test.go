package runtime

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/metadata"
)

type explicitRepo struct{}
type absentRepo struct{}
type unmarkedUserState struct {
	Repo *explicitRepo
}
type explicitController struct {
	Repo  *explicitRepo `nestgo:"inject"`
	Extra *absentRepo   `nestgo:"-"`
}
type explicitOptional struct {
	Extra *absentRepo `nestgo:"optional"`
}
type explicitRequired struct {
	Missing *absentRepo `nestgo:"inject"`
}
type explicitAtomic struct {
	Repo    *explicitRepo `nestgo:"inject"`
	Missing *absentRepo   `nestgo:"inject"`
}
type explicitLoggerA struct{}
type explicitLoggerB struct{}

func (*explicitLoggerA) String() string { return "a" }
func (*explicitLoggerB) String() string { return "b" }

type explicitInterface struct {
	Printer fmt.Stringer `nestgo:"inject"`
}
type explicitIgnored struct {
	Repo *explicitRepo `nestgo:"-"`
}
type explicitUnknown struct {
	Repo *explicitRepo `nestgo:"unexpected"`
}
type explicitUnexported struct {
	repo *explicitRepo `nestgo:"inject"`
}
type explicitNonPointer struct {
	Count int `nestgo:"inject"`
}

func TestP0UnmarkedPointersAreNotDependencies(t *testing.T) {
	target := &unmarkedUserState{}
	repoType := reflect.TypeOf((*explicitRepo)(nil))
	if err := injectFields(reflect.ValueOf(target), map[reflect.Type]reflect.Value{
		repoType: reflect.ValueOf(&explicitRepo{}),
	}, []reflect.Type{repoType}, nil); err != nil {
		t.Fatalf("unmarked user field must not be injected: %v", err)
	}
	if target.Repo != nil {
		t.Fatal("unmarked field should remain nil even if a matching provider exists")
	}
	if err := injectFields(reflect.ValueOf(&unmarkedUserState{}), nil, nil, nil); err != nil {
		t.Fatalf("unmarked field must not require a registered provider: %v", err)
	}
}

func TestP0InjectionMarkersAndControllerConstruction(t *testing.T) {
	repoType := reflect.TypeOf((*explicitRepo)(nil))
	cType := reflect.TypeOf((*explicitController)(nil))
	graph, err := prepareBootstrap([]metadata.Package{{
		Providers:   []metadata.Provider{{Type: repoType}},
		Controllers: []metadata.Controller{{Type: cType}},
	}}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	controller, err := graph.newController(cType)
	if err != nil {
		t.Fatal(err)
	}
	result := controller.Interface().(*explicitController)
	if result.Repo == nil || result.Extra != nil {
		t.Fatalf("incorrect required/ignored injection: %+v", result)
	}
}

func TestP0InjectionValidationIsAtomic(t *testing.T) {
	repoType := reflect.TypeOf((*explicitRepo)(nil))
	repo := &explicitRepo{}
	target := &explicitAtomic{}
	err := injectFields(reflect.ValueOf(target), map[reflect.Type]reflect.Value{
		repoType: reflect.ValueOf(repo),
	}, []reflect.Type{repoType}, nil)
	if err == nil || !strings.Contains(err.Error(), "missing required dependency") {
		t.Fatalf("expected missing dependency, got %v", err)
	}
	if target.Repo != nil {
		t.Fatal("injected first field despite failed preflight")
	}
}

func TestP0InjectionEdgeCases(t *testing.T) {
	repoType := reflect.TypeOf((*explicitRepo)(nil))
	repo := reflect.ValueOf(&explicitRepo{})
	aType := reflect.TypeOf((*explicitLoggerA)(nil))
	bType := reflect.TypeOf((*explicitLoggerB)(nil))
	for _, tc := range []struct {
		name      string
		instance  any
		providers map[reflect.Type]reflect.Value
		order     []reflect.Type
		want      string
	}{
		{"optional missing", &explicitOptional{}, nil, nil, ""},
		{"required missing", &explicitRequired{}, nil, nil, "missing required dependency"},
		{"explicit exclusion", &explicitIgnored{}, map[reflect.Type]reflect.Value{repoType: repo}, []reflect.Type{repoType}, ""},
		{"typed nil pointer", &explicitController{}, map[reflect.Type]reflect.Value{repoType: reflect.ValueOf((*explicitRepo)(nil))}, []reflect.Type{repoType}, "nil dependency"},
		{"interface ambiguity", &explicitInterface{}, map[reflect.Type]reflect.Value{aType: reflect.ValueOf(&explicitLoggerA{}), bType: reflect.ValueOf(&explicitLoggerB{})}, []reflect.Type{aType, bType}, "ambiguous dependency"},
		{"typed nil interface provider", &explicitInterface{}, map[reflect.Type]reflect.Value{aType: reflect.ValueOf((*explicitLoggerA)(nil))}, []reflect.Type{aType}, "nil dependency"},
		{"unknown marker", &explicitUnknown{}, nil, nil, "invalid nestgo injection tag"},
		{"unexported tagged field", &explicitUnexported{}, nil, nil, "cannot inject inaccessible field"},
		{"non pointer tagged field", &explicitNonPointer{}, nil, nil, "cannot inject non-pointer field"},
		{"invalid injection target", (*explicitRepo)(nil), nil, nil, "invalid injection target"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := injectFields(reflect.ValueOf(tc.instance), tc.providers, tc.order, nil)
			if tc.want == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("got %v want %q", err, tc.want)
			}
		})
	}
}

func TestP0InterfaceInjectionScopedVisibility(t *testing.T) {
	aType := reflect.TypeOf((*explicitLoggerA)(nil))
	bType := reflect.TypeOf((*explicitLoggerB)(nil))
	target := &explicitInterface{}
	targetType := reflect.TypeOf(target)
	providers := map[reflect.Type]reflect.Value{aType: reflect.ValueOf(&explicitLoggerA{}), bType: reflect.ValueOf(&explicitLoggerB{})}
	err := injectFields(reflect.ValueOf(target), providers, []reflect.Type{aType, bType}, map[reflect.Type]map[reflect.Type]bool{
		targetType: {aType: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if target.Printer == nil || target.Printer.String() != "a" {
		t.Fatalf("wrong scoped dependency: %v", target.Printer)
	}
}

func TestP0InjectionErrorHasStableContext(t *testing.T) {
	target := &explicitRequired{}
	err := injectFields(reflect.ValueOf(target), nil, nil, nil)
	var injection *InjectionError
	if !errors.As(err, &injection) {
		t.Fatalf("expected structured InjectionError, got %T: %v", err, err)
	}
	if injection.Code != "missing_dependency" ||
		injection.Target != reflect.TypeOf(*target) ||
		injection.Field != "Missing" ||
		injection.Dependency != reflect.TypeOf((*absentRepo)(nil)) {
		t.Fatalf("wrong injection context: %+v", injection)
	}

	_, err = prepareBootstrap([]metadata.Package{{
		Providers: []metadata.Provider{{Type: reflect.TypeOf((*explicitRequired)(nil))}},
	}}, nil, nil)
	if !errors.As(err, &injection) || injection.Code != "missing_dependency" {
		t.Fatalf("bootstrap must preserve structured DI failure: %v", err)
	}
}
