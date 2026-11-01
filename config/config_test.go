package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadAndValidate(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("APP_HOST=local\nAPP_PORT=1234\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_HOST", "override")
	s, err := Load(Options{EnvFiles: []string{path}})
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Host string `env:"HOST,required"`
		Port int `env:"PORT,default=5432,min=1,max=65535"`
		Timeout time.Duration `env:"TIMEOUT,default=3s"`
	}
	if err := s.Decode("APP", &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "override" || cfg.Port != 1234 || cfg.Timeout != 3*time.Second {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestValidationRejectsInvalidValuesWithoutLeakingSecrets(t *testing.T) {
	t.Setenv("DB_PASSWORD", "secret-value")
	t.Setenv("DB_PORT", "99999")
	s, err := Load(Options{IgnoreEnvFile: true})
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Password string `env:"PASSWORD,required"`
		Host string `env:"HOST,required"`
		Port int `env:"PORT,min=1,max=65535"`
	}
	err = s.Decode("DB", &cfg)
	if err == nil || !strings.Contains(err.Error(), "DB_HOST") || strings.Contains(err.Error(), "secret-value") {
		t.Fatalf("incorrect sanitized validation error: %v", err)
	}
}
