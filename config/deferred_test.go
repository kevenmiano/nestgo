package config

import "testing"

func TestDeferredEnvironmentSnapshotsOnlyOnFirstRead(t *testing.T) {
	const key = "NESTGO_DEFERRED_ENV_TEST"
	t.Setenv(key, "before")
	settings := DeferredEnvironment()
	t.Setenv(key, "first-read")
	if value, ok := settings.Get(key); !ok || value != "first-read" {
		t.Fatalf("first snapshot returned (%q, %v)", value, ok)
	}
	t.Setenv(key, "later")
	if value, ok := settings.Get(key); !ok || value != "first-read" {
		t.Fatalf("snapshot mutated after first read: (%q, %v)", value, ok)
	}
}

func TestDeferredEnvironmentDecodeAndNilAccess(t *testing.T) {
	t.Setenv("NESTGO_DEFERRED_PORT", "5544")
	settings := DeferredEnvironment()
	var env struct {
		Port int `env:"PORT,required,min=1"`
	}
	if err := settings.Decode("NESTGO_DEFERRED", &env); err != nil {
		t.Fatal(err)
	}
	if env.Port != 5544 {
		t.Fatalf("decoded port=%d", env.Port)
	}
	var unset *Service
	if value, ok := unset.Get("PORT"); ok || value != "" {
		t.Fatalf("nil Get = (%q, %v)", value, ok)
	}
}
