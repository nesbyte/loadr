package core

import (
	"html/template"
	"io/fs"
)

func NewTemplateContext[T any](baseConfig BaseConfig, baseData T, basePatterns ...string) *TemplateContext[T] {
	return &TemplateContext[T]{
		templateContextCore: templateContextCore{
			config:        &baseConfig,
			baseTemplates: basePatterns,
			funcMap:       &template.FuncMap{},
		},
		baseData: &baseData,
	}
}

// The Base render is the main data structure
// which the templates are using internally for their rendering through
// composition.
type TemplateContext[T any] struct {
	templateContextCore
	baseData *T
}

type templateContextCore struct {
	config        *BaseConfig
	baseTemplates []string // The base templates that are used and settable
	withTemplates []string
	onLoad        func() error      // If set, called before the templates are loaded
	funcMap       *template.FuncMap // Functions that will be added to the templates
}

// Performs a shallow copy equivalent of TemplateContext
// but where the TemplateContext and WithTemplates
// are cloned to allow for overriding the templates
// without changing the original TemplateContext.
//
// Changes in the Config and BaseData will propegate
// to the copied TemplateContext.
func (tc *TemplateContext[T]) Copy(patterns ...string) *TemplateContext[T] {
	bt := append([]string(nil), tc.baseTemplates...)
	at := append([]string(nil), tc.withTemplates...)
	newTemplateContext := TemplateContext[T]{
		templateContextCore: templateContextCore{
			config:        tc.config,
			baseTemplates: bt,
			withTemplates: at,
			funcMap:       tc.funcMap,
		},
		baseData: tc.baseData,
	}

	return &newTemplateContext

}

type BaseConfig struct {
	FS fs.FS // Sets the FS of the renderer, us fs.Sub to specify root of the FS
}

// Sets the configuration of the BaseTemplates
// If SetConfig is called multiple times on the same
// base render, the last call is used
func (tc *TemplateContext[T]) SetConfig(config BaseConfig) *TemplateContext[T] {
	*tc.config = config
	return tc
}

func (tc TemplateContext[T]) Config() BaseConfig {
	return *tc.config
}

// Sets the data which will be passed in on every
// Render() call.
// If SetBaseData is called multiple times on the same TemplateContext, the last
// call is used.
// This works immediately and independently from calling loadr.LoadTemplates()
func (tc *TemplateContext[T]) SetBaseData(data T) *TemplateContext[T] {
	*tc.baseData = data
	return tc
}

// Sets all the templates to be parsed
// SetTemplates overwrites previous SetTemplates calls
func (tc *TemplateContext[T]) SetBaseTemplates(patterns ...string) *TemplateContext[T] {
	tc.baseTemplates = patterns
	return tc
}

func (tc *TemplateContext[T]) SetWithTemplates(patterns ...string) *TemplateContext[T] {
	tc.withTemplates = patterns
	return tc
}

// The same as Copy().SetWithTemplates()
// Copies and adds templates which will be parsed together with the base templates.
// Useful when you have a root index page whith multiple sub-templates
// with the same template name defined.
func (tc *TemplateContext[T]) WithTemplates(patterns ...string) *TemplateContext[T] {
	tcc := tc.Copy()
	tcc.SetWithTemplates(patterns...)
	return tcc
}

// Alias for AddTemplates, it can make the code more readable
func (tc *TemplateContext[T]) WT(patterns ...string) *TemplateContext[T] {
	return tc.WithTemplates(patterns...)
}

// Sets the onLoad function which will be called
// before every template (using NewTemplate and loadr.Load()) is loaded.
// As an example, cache busting logic can be implemented here from manifest files
// and then passed in to the tempaltes using SetBaseTemplates()
func (tc *TemplateContext[T]) SetOnTemplateLoad(onLoad func() error) {
	// Currently inefficient as it is run every time on load
	// if there are many templates it runs multiple times
	tc.onLoad = onLoad
}

// Adds the FuncMap functions to the template context using the
// std template.FuncMap type
func (tc *TemplateContext[T]) Funcs(funcMap template.FuncMap) *TemplateContext[T] {
	*tc.funcMap = funcMap
	return tc
}
