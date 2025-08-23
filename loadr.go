package loadr

import (
	"context"
	"log"
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

// Lazily prepares the templates for loading by pattern both file names
// as well as the template names can be provided. If no template name is provided,
// the template name will be the first name of the SetTemplates() pattern
//
// The expected data structure should also be provided as it is used
// for the loading and validation when loadr.LoadTemplates() is called.
//
// No templates get parsed until loadr.Validate() is run
func NewTemplate[T, U any](tc *core.TemplateContext[T], pattern string, data U) *core.Templ[T, U] {
	return core.NewTemplate(tc, pattern, data)
}

func NewSubTemplate[T, U any](tc *core.TemplateContext[T], pattern string, data U) *core.Templ[T, U] {
	return core.NewSubTemplate(tc, pattern, data)
}

// Loads and validates all the created templates.
// It is expected to be called after all the templates and settings have been created
func LoadTemplates() error {
	return registry.LoadTemplates()
}

// Watches the specified local pathsToWatch for file changes and notifies connected clients
// and handleChange if provided.
//
// Live reload can only be started once.
//
// The handlePattern is the URL path that the live server will handle and must match the
// registered pattern in the HTTP server.
// If no handlePatern is provided, the live server will serve on /live-server
func RunLiveReload(handlePattern string, handleReload func(fsnotify.Event, error), pathsToWatch ...string) (http.HandlerFunc, context.CancelFunc, error) {
	return livereload.RunLiveReload(handlePattern, handleReload, pathsToWatch...)
}

// A basic helper function for LiveReload to perform logging when a reload occurs
func HandleReload(e fsnotify.Event, err error) {
	if err == nil {
		log.Println("reloaded", e.Name)
	} else {
		log.Println("error:", err.Error())
	}
}
