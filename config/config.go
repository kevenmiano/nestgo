// Package config loads and validates typed environment configuration at bootstrap.
package config

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Options controls dotenv loading. Environment variables take precedence over files.
type Options struct {
	EnvFiles []string
	IgnoreEnvFile bool
}

// Service exposes an immutable snapshot of the resolved configuration.
type Service struct {
	values map[string]string
}

func Load(opts Options) (*Service, error) {
	values := make(map[string]string)
	if !opts.IgnoreEnvFile {
		for _, path := range opts.EnvFiles {
			if err := readEnv(path, values); err != nil {
				return nil, err
			}
		}
	}
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}
	return &Service{values: values}, nil
}

func readEnv(path string, values map[string]string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("config: read %s: %w", path, err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return fmt.Errorf("config: invalid entry in %s", path)
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"' || value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1:len(value)-1]
		}
		if _, exists := values[key]; !exists {
			values[key] = value
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("config: scan %s: %w", path, err)
	}
	return nil
}

func (s *Service) Get(key string) (string, bool) {
	if s == nil {
		return "", false
	}
	v, ok := s.values[key]
	return v, ok
}

// Decode validates configuration once at startup and assigns a typed struct.
// Tags: env:"KEY,required" or env:"KEY,default=value,min=1,max=65535".
// Fields support string, bool, signed integers and time.Duration.
func (s *Service) Decode(prefix string, target any) error {
	if s == nil {
		return fmt.Errorf("config: nil service")
	}
	rv := reflect.ValueOf(target)
	if !rv.IsValid() || rv.Kind() != reflect.Ptr || rv.IsNil() || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("config: target must be pointer to struct")
	}
	dst := rv.Elem()
	typ := dst.Type()
	var problems []string
	for i := 0; i < dst.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("env")
		if tag == "" {
			continue
		}
		if !dst.Field(i).CanSet() {
			problems = append(problems, field.Name+": field not exported")
			continue
		}
		parts := strings.Split(tag, ",")
		name := parts[0]
		if name == "" {
			name = strings.ToUpper(field.Name)
		}
		key := name
		if prefix != "" {
			key = prefix+"_"+name
		}
		value, exists := s.Get(key)
		required := false
		min, max := "", ""
		for _, setting := range parts[1:] {
			switch {
			case setting == "required":
				required = true
			case strings.HasPrefix(setting, "default="):
				if !exists {
					value = strings.TrimPrefix(setting, "default=")
					exists = true
				}
			case strings.HasPrefix(setting, "min="):
				min = strings.TrimPrefix(setting, "min=")
			case strings.HasPrefix(setting, "max="):
				max = strings.TrimPrefix(setting, "max=")
			default:
				problems = append(problems, key+": unknown validation "+setting)
			}
		}
		if required && (!exists || value == "") {
			problems = append(problems, key+": required")
			continue
		}
		if !exists {
			continue
		}
		dest := dst.Field(i)
		switch {
		case dest.Type() == reflect.TypeOf(time.Duration(0)):
			parsed, err := time.ParseDuration(value)
			if err != nil {
				problems = append(problems, key+": invalid duration")
			} else {
				dest.SetInt(int64(parsed))
			}
		case dest.Kind() == reflect.String:
			dest.SetString(value)
		case dest.Kind() == reflect.Bool:
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				problems = append(problems, key+": invalid boolean")
			} else {
				dest.SetBool(parsed)
			}
		case dest.Kind() >= reflect.Int && dest.Kind() <= reflect.Int64:
			parsed, err := strconv.ParseInt(value, 10, dest.Type().Bits())
			if err != nil {
				problems = append(problems, key+": invalid integer")
				continue
			}
			if min != "" {
				floor, err := strconv.ParseInt(min, 10, 64)
				if err != nil || parsed < floor {
					problems = append(problems, key+": below minimum or invalid min")
				}
			}
			if max != "" {
				ceiling, err := strconv.ParseInt(max, 10, 64)
				if err != nil || parsed > ceiling {
					problems = append(problems, key+": above maximum or invalid max")
				}
			}
			dest.SetInt(parsed)
		default:
			problems = append(problems, key+": unsupported field type")
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("config: validation failed: %s", strings.Join(problems, "; "))
	}
	return nil
}
