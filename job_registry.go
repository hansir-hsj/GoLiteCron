package golitecron

import (
	"fmt"
	"os"
	"sync"
)

var (
	jobRegistry = make(map[string]any) // func() error or func(context.Context) error
	registryMu  sync.RWMutex
)

// RegisterJob registers a job function by name.
// Accepts func() error or func(context.Context) error.
// If a job with the same name already exists, it will be overwritten and a warning will be printed to stderr.
func RegisterJob(name string, fn any) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := jobRegistry[name]; exists {
		fmt.Fprintf(os.Stderr, "Warning: job '%s' already registered, overwriting previous registration\n", name)
	}
	jobRegistry[name] = fn
}

// GetJob retrieves a registered job function by name.
// Returns func() error or func(context.Context) error.
func GetJob(name string) (any, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	fn, ok := jobRegistry[name]
	return fn, ok
}
