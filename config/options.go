package config

import (
	"errors"
	"strings"
)

// Option configures config.New. File parsing remains delegated to the selected
// loader; the base package provides a small, backward-compatible default.
type Option func(*Options) error

func New(opts ...Option) (*Service, error) {
	options := Options{}
	for _, option := range opts {
		if option == nil {
			return nil, errors.New("config: nil option")
		}
		if err := option(&options); err != nil {
			return nil, err
		}
	}
	return Load(options)
}

func WithEnvFile(paths ...string) Option {
	return func(o *Options) error {
		for _, path := range paths {
			if strings.TrimSpace(path) == "" {
				return errors.New("config: empty env filename")
			}
		}
		o.EnvFiles = append(o.EnvFiles, paths...)
		return nil
	}
}

func WithIgnoreEnvFile() Option {
	return func(o *Options) error {
		o.IgnoreEnvFile = true
		return nil
	}
}
