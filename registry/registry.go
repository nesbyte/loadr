package registry

import (
	"sync"
)

type Loader interface {
	Load() error
}

// Use any as the map type and then
// infer our the interface from it
type registry struct {
	loaders    map[Loader]struct{}
	liveReload bool   // If true, sets the Templ to reload on every Render() call
	jsToInject string // JS to inject at the end of the body
}

// the U must be any as the data is not set
// the T should be a generic however
// it is unclear on how to instantiate the
// object without knowing T and especially
// as different objects may have different types for T
var store = registry{loaders: make(map[Loader]struct{})}

var mu sync.Mutex

// Adds a BaseRender and it's pattern to the register
func Add(l Loader) {
	_, ok := store.loaders[l]
	mu.Lock()
	if !ok {
		store.loaders[l] = struct{}{}
	}
	mu.Unlock()
}

// Enables or disables live reloading
func SetLiveReload(enabled bool) {
	store.liveReload = enabled
}

// Checks if live reloading is enabled
func LiveReload() bool {
	return store.liveReload
}

func SetJSToInject(b []byte) {
	store.jsToInject = string(b)
}

func JSToInject() string {
	return store.jsToInject
}

// Prepares the templates by loading and validating them
func LoadTemplates() error {

	for loader := range store.loaders {
		err := loader.Load()
		if err != nil {
			return err
		}
	}

	return nil
}
