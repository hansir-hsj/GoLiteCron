package golitecron

import "sync"

var (
	jobRegistry = make(map[string]func() error)
	registryMu  sync.RWMutex
)

func RegisterJob(name string, fn func() error) {
	registryMu.Lock()
	defer registryMu.Unlock()
	jobRegistry[name] = fn
}

func GetJob(name string) (func() error, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	fn, ok := jobRegistry[name]
	return fn, ok
}
