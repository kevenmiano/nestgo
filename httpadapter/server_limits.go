package httpadapter

import "time"

// ServerLimits bounds the resources consumed by an HTTP connection.
// Zero and negative overrides retain the secure defaults; limits cannot be
// disabled accidentally through an empty configuration.
type ServerLimits struct {
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	MaxBodyBytes      int
}

// DefaultServerLimits applies to all built-in transport adapters.
func DefaultServerLimits() ServerLimits {
	return ServerLimits{
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    16 << 10,
		MaxBodyBytes:      1 << 20,
	}
}

// Override applies only strictly positive fields, preserving secure defaults.
func (l ServerLimits) Override(other ServerLimits) ServerLimits {
	if other.ReadHeaderTimeout > 0 {
		l.ReadHeaderTimeout = other.ReadHeaderTimeout
	}
	if other.ReadTimeout > 0 {
		l.ReadTimeout = other.ReadTimeout
	}
	if other.WriteTimeout > 0 {
		l.WriteTimeout = other.WriteTimeout
	}
	if other.IdleTimeout > 0 {
		l.IdleTimeout = other.IdleTimeout
	}
	if other.MaxHeaderBytes > 0 {
		l.MaxHeaderBytes = other.MaxHeaderBytes
	}
	if other.MaxBodyBytes > 0 {
		l.MaxBodyBytes = other.MaxBodyBytes
	}
	return l
}
