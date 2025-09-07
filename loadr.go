package loadr

import (
	"net/http"

	"github.com/fsnotify/fsnotify"
	"github.com/nesbyte/loadr/core"
	"github.com/nesbyte/loadr/livereload"
	"github.com/nesbyte/loadr/registry"
)

// Creates a new base template with the provided baseData.
//
// All NewTemplateContext calls will use TemplateContext as their starting point.
//
// The baseData is used to define the data type passed in to the
// template for the base data for all child templates.
func NewTemplateContext[T any](config BaseConfig, baseData T, basePatterns ...string) *core.TemplateContext[T] {
	return core.NewTemplateContext(config, baseData, basePatterns...)
}

// Used to set the configuration for the base templates
type BaseConfig = core.BaseConfig

const NoData = 0

// Lazily prepares the base template (the first template name provided in the basePattern of NewTemplateContext).
// Base data as well as Render data will be passed in on Render(w, data) call as .B and .D respectively.
//
// The expected data structure which will be used by the Render(w, data) method should also be provided as it is used
// for the loading and validation when loadr.LoadTemplates() is called.
//
// No templates get parsed until loadr.Validate() is run
func NewTemplate[T, U any](tc *core.TemplateContext[T], data U) *core.Template[T, U] {
	return core.NewTemplate(tc, data)
}

// Similar to NewTemplate, but allows a template to be created
// that matches the provided pattern. The returned template
// does not include base data when Render(*,*) is called, hence also does not rely on .B and .D
//
// No templates get parsed until loadr.Validate() is run
func NewSubTemplate[T, U any](tc *core.TemplateContext[T], pattern string, data U) *core.SubTemplate[U] {
	return core.NewSubTemplate(tc, pattern, data)
}

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
