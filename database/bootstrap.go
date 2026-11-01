package database

import (
	"context"
	"errors"
	"fmt"
)

// Bootstrap opens all configured named connections. On any failure it closes
// previously opened pools, preventing a partially initialized application.
func Bootstrap(ctx context.Context, options []ConnectionOptions) (*Connections, error) {
	registry := NewConnections()
	seen := make(map[string]bool, len(options))
	for _, option := range options {
		if option.Name == "" || seen[option.Name] {
			return nil, errors.Join(fmt.Errorf("database: empty or duplicate connection name %q", option.Name), registry.Close())
		}
		seen[option.Name] = true
		if err := registry.Open(ctx, option); err != nil {
			return nil, errors.Join(err, registry.Close())
		}
	}
	return registry, nil
}
