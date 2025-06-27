package core

import (
	"io/fs"
)

func NewBaseTemplate[T any](baseData T) *BaseTemplates[T] {
	return &BaseTemplates[T]{baseData: &baseData}
}

// The Base render is the main data structure
// which the templates are using internally for their rendering through
// composition.
type BaseTemplates[T any] struct {
	config        *BaseConfig
	baseData      *T
	baseTemplates []string // The base templates that are used and settable
	withTemplates []string
	onLoad        func() error // If set, called before the templates are loaded
}

// Performs a shallow copy equivalent of BaseTemplates
// but where the BaseTemplates and WithTemplates
// are cloned to allow for overriding the templates
// without changing the original BaseTemplates.
//
// Changes in the Config and BaseData will propegate
// to the copied BaseTemplates.
func (b *BaseTemplates[T]) Copy(patterns ...string) *BaseTemplates[T] {
	bt := append([]string(nil), b.baseTemplates...)
	at := append([]string(nil), b.withTemplates...)
	newBaseTemplates := BaseTemplates[T]{
		config:        b.config,
		baseData:      b.baseData,
		baseTemplates: bt,
		withTemplates: at,
	}

	return &newBaseTemplates

}

type BaseConfig struct {
	FS fs.FS // Sets the FS of the renderer, us fs.Sub to specify root of the FS
}

// Sets the configuration of the BaseTemplates
// If SetConfig is called multiple times on the same
// base render, the last call is used
func (b *BaseTemplates[T]) SetConfig(config BaseConfig) *BaseTemplates[T] {
	b.config = &config
	return b
}

func (b BaseTemplates[T]) Config() BaseConfig {
	return *b.config
}

// Sets the data which will be passed in on every
// Render() call.
// If SetBaseData is called multiple times on the same BaseTemplates, the last
// call is used.
// This works immediately and independently from calling loadr.LoadTemplates()
func (b *BaseTemplates[T]) SetBaseData(data T) *BaseTemplates[T] {
	b.baseData = &data
	return b
}

// Sets all the templates to be parsed
// SetTemplates overwrites previous SetTemplates calls
func (b *BaseTemplates[T]) SetBaseTemplates(patterns ...string) *BaseTemplates[T] {
	b.baseTemplates = patterns
	return b
}

// Copies and adds templates which will be parsed together with the base templates.
// Becomes useful when you have a root index page whith multiple sub-templates
// with the same template name defined.
func (b *BaseTemplates[T]) WithTemplates(patterns ...string) *BaseTemplates[T] {
	c := b.Copy()
	c.withTemplates = patterns
	return b
}

// Alias for AddTemplates, it can make the code more readable
func (b *BaseTemplates[T]) WT(patterns ...string) *BaseTemplates[T] {
	return b.WithTemplates(patterns...)
}

// Sets the onLoad function which will be called
// before every template (using NewTemplate and loadr.Load()) is loaded.
// As an example, cache busting logic can be implemented here from manifest files
// and then passed in to the tempaltes using SetBaseTemplates()
func (b *BaseTemplates[T]) SetOnTemplateLoad(onLoad func() error) {
	// Currently inefficient as it is run every time on load
	// if there are many templates it runs multiple times
	b.onLoad = onLoad
}
