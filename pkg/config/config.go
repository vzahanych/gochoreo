package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// Options defines how configuration should be discovered and loaded.
type Options struct {
	// ConfigFile is an explicit path to the configuration file. If provided,
	// Name/Type/Paths are ignored.
	ConfigFile string

	// Name is the base name (without extension) of the config file, e.g. "gateway".
	Name string

	// Type is the configuration file type (yaml, yml, json, toml, etc.). Optional.
	Type string

	// Paths are directories to search for the configuration file, in order.
	Paths []string

	// Required indicates whether a missing config file should be treated as an error.
	Required bool

	// EnvPrefix, if set, is used as a prefix for environment variables (e.g. APP_...).
	EnvPrefix string

	// AutomaticEnv enables automatic environment variable binding.
	AutomaticEnv bool

	// EnvKeyReplacer replaces characters in key names when binding env vars.
	// If nil and AutomaticEnv is true, a default replacer (".", "-" -> "_") is used.
	EnvKeyReplacer *strings.Replacer

	// Defaults sets default values per dotted key path (e.g. "server.port": 8080).
	Defaults map[string]any

	// TagName controls which struct tag viper/mapstructure should use (default: "yaml").
	TagName string
}

// Loader wraps a configured viper instance and options.
type Loader struct {
	v    *viper.Viper
	opts Options
}

// NewLoader creates a new Loader with the provided options.
func NewLoader(opts Options) *Loader {
	if opts.TagName == "" {
		opts.TagName = "yaml"
	}

	v := viper.New()
	return &Loader{v: v, opts: opts}
}

// Viper returns the underlying viper instance for advanced use cases.
func (l *Loader) Viper() *viper.Viper { return l.v }

// LoadInto reads configuration from defaults, files, and environment variables,
// and unmarshals into the provided struct pointer. If the struct is pre-populated
// with defaults, those will be preserved and only overridden by loaded values.
func (l *Loader) LoadInto(ctx context.Context, out any) error { // ctx reserved for future hooks
	// 1) Defaults
	for k, v := range l.opts.Defaults {
		l.v.SetDefault(k, v)
	}

	// 2) File
	if l.opts.ConfigFile != "" {
		l.v.SetConfigFile(l.opts.ConfigFile)
	} else {
		if l.opts.Name != "" {
			l.v.SetConfigName(l.opts.Name)
		}
		if l.opts.Type != "" {
			l.v.SetConfigType(l.opts.Type)
		}
		for _, p := range l.opts.Paths {
			l.v.AddConfigPath(p)
		}
	}

	readConfig := func() error {
		if l.opts.ConfigFile == "" && l.opts.Name == "" {
			// Nothing to read; skip file step
			return nil
		}
		if err := l.v.ReadInConfig(); err != nil {
			if !l.opts.Required {
				// If not required, missing file is OK
				return nil
			}
			return fmt.Errorf("read config: %w", err)
		}
		return nil
	}

	if err := readConfig(); err != nil {
		return err
	}

	// 3) Environment
	if l.opts.AutomaticEnv {
		if l.opts.EnvKeyReplacer != nil {
			l.v.SetEnvKeyReplacer(l.opts.EnvKeyReplacer)
		} else {
			// Replace dots and dashes with underscores for env var compatibility
			l.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
		}
		if l.opts.EnvPrefix != "" {
			l.v.SetEnvPrefix(l.opts.EnvPrefix)
		}
		l.v.AutomaticEnv()
	}

	// 4) Unmarshal into the provided struct pointer
	decodeHook := mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	)

	if err := l.v.Unmarshal(out, viper.DecodeHook(decodeHook)); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}

	return nil
}
