package runtime

import "time"

type BootstrapMetrics struct {
	RouteRegistration time.Duration
	LifecycleInitialization time.Duration
	Total time.Duration
}
