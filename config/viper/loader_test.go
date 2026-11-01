package viper

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kevenmiano/nestgo/config"
)

func TestLoadDotEnvAndOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("DB_HOST=local\nDB_PORT=5432\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DB_HOST", "override")
	service, err := Load(config.Options{EnvFiles: []string{path}})
	if err != nil {
		t.Fatal(err)
	}
	var settings struct {
		Host string `env:"HOST,required"`
		Port int `env:"PORT,min=1,max=65535"`
	}
	if err := service.Decode("DB", &settings); err != nil {
		t.Fatal(err)
	}
	if settings.Host != "override" || settings.Port != 5432 {
		t.Fatalf("unexpected settings: %#v", settings)
	}
}
