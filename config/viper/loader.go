// Package viper integrates the optional Viper configuration loader with NestGo.
package viper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kevenmiano/nestgo/config"
	"github.com/spf13/viper"
)

// Load reads dotenv/YAML/JSON/TOML files using Viper. Earlier files win.
// OS environment overrides file values. The result is immutable and can be
// validated once at application bootstrap through config.Service.Decode.
func Load(options config.Options) (*config.Service, error) {
	values := map[string]string{}
	if !options.IgnoreEnvFile {
		for _, filename := range options.EnvFiles {
			v := viper.New()
			v.SetConfigFile(filename)
			ext := strings.TrimPrefix(filepath.Ext(filename), ".")
			if ext == "" {
				v.SetConfigType("env")
			}
			if err := v.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("config/viper: read %s: %w", filename, err)
			}
			for _, key := range v.AllKeys() {
				upper := strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
				if _, exists := values[upper]; !exists {
					values[upper] = v.GetString(key)
				}
			}
		}
	}
	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			values[key] = value
		}
	}
	return config.FromValues(values), nil
}
