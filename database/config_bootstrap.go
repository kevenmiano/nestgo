package database

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kevenmiano/nestgo/config"
)

// ConnectionEnv maps a named pool to a validated environment prefix.
// Example prefix USERS_DB reads USERS_DB_DRIVER and USERS_DB_DSN.
type ConnectionEnv struct {
	Name string
	Prefix string
}

// BootstrapFromConfig resolves each connection using config.Service and opens
// pools atomically. Sensitive DSNs are never included in validation errors.
func BootstrapFromConfig(ctx context.Context, settings *config.Service, entries []ConnectionEnv) (*Connections, error) {
	if settings == nil {
		return nil, fmt.Errorf("database: config service required")
	}
	options := make([]ConnectionOptions, 0, len(entries))
	for _, entry := range entries {
		if entry.Name == "" || entry.Prefix == "" {
			return nil, fmt.Errorf("database: connection name and prefix required")
		}
		var env struct {
			Driver string `env:"DRIVER,required"`
			DSN string `env:"DSN,required"`
			MaxOpenConns int `env:"MAX_OPEN_CONNS,default=10,min=1"`
			MaxIdleConns int `env:"MAX_IDLE_CONNS,default=5,min=0"`
			PingTimeout time.Duration `env:"PING_TIMEOUT,default=5s"`
		}
		if err := settings.Decode(strings.TrimSuffix(entry.Prefix, "_"), &env); err != nil {
			return nil, fmt.Errorf("database: connection %s: %w", entry.Name, err)
		}
		if env.MaxIdleConns > env.MaxOpenConns {
			return nil, fmt.Errorf("database: connection %s: idle limit exceeds open limit", entry.Name)
		}
		options = append(options, ConnectionOptions{
			Name: entry.Name,
			Driver: env.Driver,
			DSN: env.DSN,
			MaxOpenConns: env.MaxOpenConns,
			MaxIdleConns: env.MaxIdleConns,
			PingTimeout: env.PingTimeout,
		})
	}
	return Bootstrap(ctx, options)
}

// ParseConnectionNames validates a comma-separated connection-name list.
// It is suitable for application bootstrap configuration.
func ParseConnectionNames(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	names := strings.Split(value, ",")
	seen := make(map[string]bool)
	for i, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			return nil, fmt.Errorf("database: empty or duplicate connection name at index %s", strconv.Itoa(i))
		}
		seen[name] = true
		names[i] = name
	}
	return names, nil
}
