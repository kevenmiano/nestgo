package database

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kevenmiano/nestgo/config"
)

// WithConnectionFromConfig resolves a named connection from a validated config
// snapshot. No sockets are opened until Module.Open is called.
func WithConnectionFromConfig(name string, settings *config.Service, prefix string) Option {
	return func(plan *Options) error {
		if settings == nil {
			return errors.New("database: config service required")
		}
		if strings.TrimSpace(name) == "" || strings.TrimSpace(prefix) == "" {
			return errors.New("database: connection name and prefix required")
		}
		var env struct {
			Driver       string        `env:"DRIVER,required"`
			DSN          string        `env:"DSN,required"`
			MaxOpenConns int           `env:"MAX_OPEN_CONNS,default=10,min=1"`
			MaxIdleConns int           `env:"MAX_IDLE_CONNS,default=5,min=0"`
			PingTimeout  time.Duration `env:"PING_TIMEOUT,default=5s"`
		}
		if err := settings.Decode(strings.TrimSuffix(prefix, "_"), &env); err != nil {
			return fmt.Errorf("database: connection %q: %w", name, err)
		}
		if env.MaxIdleConns > env.MaxOpenConns {
			return fmt.Errorf("database: connection %q: idle limit exceeds open limit", name)
		}
		plan.Connections = append(plan.Connections, ConnectionOptions{
			Name:         name,
			Driver:       env.Driver,
			DSN:          env.DSN,
			MaxOpenConns: env.MaxOpenConns,
			MaxIdleConns: env.MaxIdleConns,
			PingTimeout:  env.PingTimeout,
		})
		return nil
	}
}
