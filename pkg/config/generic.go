package config

import "context"

// LoadNew loads configuration using the provided options into a newly
// allocated instance of T. If defaults is non-nil, its values seed the
// instance before applying loaded overrides. It returns the instance
// and the Loader for advanced operations (e.g., watching).
func LoadNew[T any](ctx context.Context, opts Options, defaults *T) (*T, *Loader, error) {
	loader := NewLoader(opts)
	var instance T
	if defaults != nil {
		instance = *defaults
	}
	if err := loader.LoadInto(ctx, &instance); err != nil {
		return nil, nil, err
	}
	return &instance, loader, nil
}
