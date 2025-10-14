package loadr

import (
	"net/http"

	"github.com/fsnotify/fsnotify"
	"github.com/nesbyte/loadr/livereload"
	"github.com/nesbyte/loadr/registry"
)

// Loads and validates all the created templates.
// It must be called after all the templates and settings have been created
func LoadTemplates() error {
	return registry.LoadTemplates()
}

// The same as RunLiveReload but panics if an error occurs
func MustRunLiveReload(handlePattern string, handleReload func(fsnotify.Event, error), pathsToWatch ...string) http.HandlerFunc {
	h, err := RunLiveReload(handlePattern, handleReload, pathsToWatch...)
	if err != nil {
		panic(err)
	}
	return h
}

// Watches the specified local pathsToWatch for file changes and notifies connected clients
// and handleChange if provided.
//
// Live reload can only be started once.
//
// The handlePattern is the URL path that the live server will handle and must match the
// registered pattern in the HTTP server.
// handleReload is an optional function that will be called when a file change is detected
// and can be used for custom logging. If nil is provided a default logging function will be used.
func RunLiveReload(handlePattern string, handleReload func(fsnotify.Event, error), pathsToWatch ...string) (http.HandlerFunc, error) {
	return livereload.RunLiveReload(handlePattern, handleReload, pathsToWatch...)
}
