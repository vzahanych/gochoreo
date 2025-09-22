package config

import (
	"context"
	"fmt"

	"github.com/fsnotify/fsnotify"
)

// WatchAndReload watches the underlying config source for changes and calls the provided
// reload function with a freshly unmarshaled struct. The outFactory must return a new
// pointer to the target config struct type each time (e.g., func() *MyConfig { return &MyConfig{} }).
// The callback receives the new instance. The watch stops when the context is done.
func (l *Loader) WatchAndReload(ctx context.Context, outFactory func() any, onReload func(any)) error {
	if l.v == nil {
		return fmt.Errorf("viper instance is nil")
	}

	// Enable watching
	l.v.WatchConfig()
	l.v.OnConfigChange(func(e fsnotify.Event) {
		// Re-unmarshal into a fresh instance to avoid partial state
		fresh := outFactory()
		if err := l.LoadInto(ctx, fresh); err != nil {
			// We cannot log here generically; rely on caller to handle errors as needed
			return
		}
		onReload(fresh)
	})

	// Block until context is done
	<-ctx.Done()
	return ctx.Err()
}
