package runtime

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// OnModuleInit runs once after dependency injection and route registration.
// Implementations should not start blocking background work.
type OnModuleInit interface{ OnModuleInit(context.Context) error }

// OnApplicationBootstrap runs after all OnModuleInit hooks have succeeded.
type OnApplicationBootstrap interface{ OnApplicationBootstrap(context.Context) error }

// BeforeApplicationShutdown runs before the HTTP adapter begins shutdown.
type BeforeApplicationShutdown interface{ BeforeApplicationShutdown(context.Context) error }

// OnModuleDestroy releases provider resources after HTTP shutdown.
type OnModuleDestroy interface{ OnModuleDestroy(context.Context) error }

// OnApplicationShutdown runs at the end of application shutdown.
type OnApplicationShutdown interface{ OnApplicationShutdown(context.Context) error }

type lifecycleRegistry struct {
	onProvider   func(reflect.Type)
	onController func(reflect.Type, string)
	onRoute      func(string, string)
	instances    []any
	shutdownOnce sync.Once
	shutdownErr  error
}

func (r *lifecycleRegistry) initialize(ctx context.Context) error {
	for _, instance := range r.instances {
		if err := ctx.Err(); err != nil {
			return err
		}
		if h, ok := instance.(OnModuleInit); ok {
			if err := h.OnModuleInit(ctx); err != nil {
				return fmt.Errorf("OnModuleInit %T: %w", instance, err)
			}
		}
	}
	for _, instance := range r.instances {
		if err := ctx.Err(); err != nil {
			return err
		}
		if h, ok := instance.(OnApplicationBootstrap); ok {
			if err := h.OnApplicationBootstrap(ctx); err != nil {
				return fmt.Errorf("OnApplicationBootstrap %T: %w", instance, err)
			}
		}
	}
	return ctx.Err()
}

func (r *lifecycleRegistry) shutdown(ctx context.Context, closeAdapter func(context.Context) error) error {
	r.shutdownOnce.Do(func() {
		var errs []error
		for i := len(r.instances) - 1; i >= 0; i-- {
			if h, ok := r.instances[i].(BeforeApplicationShutdown); ok {
				if err := h.BeforeApplicationShutdown(ctx); err != nil {
					errs = append(errs, fmt.Errorf("BeforeApplicationShutdown %T: %w", r.instances[i], err))
				}
			}
		}
		if err := closeAdapter(ctx); err != nil {
			errs = append(errs, err)
		}
		for i := len(r.instances) - 1; i >= 0; i-- {
			if h, ok := r.instances[i].(OnModuleDestroy); ok {
				if err := h.OnModuleDestroy(ctx); err != nil {
					errs = append(errs, fmt.Errorf("OnModuleDestroy %T: %w", r.instances[i], err))
				}
			}
		}
		for i := len(r.instances) - 1; i >= 0; i-- {
			if h, ok := r.instances[i].(OnApplicationShutdown); ok {
				if err := h.OnApplicationShutdown(ctx); err != nil {
					errs = append(errs, fmt.Errorf("OnApplicationShutdown %T: %w", r.instances[i], err))
				}
			}
		}
		r.shutdownErr = errors.Join(errs...)
	})
	return r.shutdownErr
}
