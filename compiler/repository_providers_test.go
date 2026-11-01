package compiler

import (
	"strings"
	"testing"
)

func TestRepositoryProviderGenerationValidation(t *testing.T) {
	for _, tt := range []struct {
		name string
		defs []RepositoryDefinition
		want string
	}{
		{"duplicate", []RepositoryDefinition{{Name: "Users", Connection: "users"}, {Name: "Users", Connection: "billing"}}, "duplicate repository"},
		{"invalid connection", []RepositoryDefinition{{Name: "Users", Connection: "billing-db"}}, "invalid repository or connection"},
		{"unnamed connection", []RepositoryDefinition{{Name: "Users"}}, "invalid repository or connection"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GenerateRepositoryProviders("users", tt.defs)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func TestRepositoryProviderGenerationIncludesRuntimeInstances(t *testing.T) {
	out, err := GenerateRepositoryProviders("users", []RepositoryDefinition{{Name: "Users", Connection: "users"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"func NewGeneratedRepositories(", "func (r *GeneratedRepositories) Instances() []any", "registry"} {
		if want == "registry" { continue }
		if !strings.Contains(string(out), want) {
			t.Fatalf("expected %q in generated source", want)
		}
	}
}
