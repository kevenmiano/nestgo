package runtime

import (
	"context"
	"testing"
)

func TestCodeFirstBootstrapOwnsLifecycleAndIndependentServices(t *testing.T) {
	const key = "NESTGO_BOOTSTRAP_LAZY_ENV"
	t.Setenv(key, "first")

	first, err := New()
	if err != nil {
		t.Fatal(err)
	}
	second, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if first.lifecycle != &first.lifecycleState || second.lifecycle != &second.lifecycleState {
		t.Fatal("application lifecycle owner must live inside its application")
	}
	if first.lifecycle == second.lifecycle {
		t.Fatal("different applications must not share lifecycle state")
	}
	if first.Config == nil || first.Connections == nil || second.Config == nil || second.Connections == nil {
		t.Fatal("unconfigured applications must still expose configuration and SQL services")
	}
	if first.Config == second.Config || first.Connections == second.Connections {
		t.Fatal("unconfigured services must be independently owned by each application")
	}
	t.Setenv(key, "after")
	if got, ok := first.Config.Get(key); !ok || got != "after" {
		t.Fatalf("lazy environment first read: %q, %v", got, ok)
	}
	t.Setenv(key, "later")
	if got, _ := first.Config.Get(key); got != "after" {
		t.Fatalf("service snapshot changed after first read: %q", got)
	}
	if got, _ := second.Config.Get(key); got != "later" {
		t.Fatalf("second application's snapshot not independent: %q", got)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if err := first.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
}
