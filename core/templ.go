package core

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/nesbyte/loadr/livereload"
	"github.com/nesbyte/loadr/registry"
)

var ErrTemplateExecute = errors.New("template execute error")

// TemplateError is the error type returned by the template loading and rendering
// functions. It wraps the underlying error and provides context about the
// template patterns used.
type TemplateError struct {
	ctx        templateContextCore
	usePattern string
	Err        error
}

func (e TemplateError) Error() string {
	return fmt.Sprintf("basetemplates %q with templates %q and template pattern %q failed: %s", e.ctx.baseTemplates, strings.Join(e.ctx.withTemplates, ", "), e.usePattern, e.Err.Error())
}

func (e TemplateError) Unwrap() error {
	return e.Err
}

type Template[T, U any] struct {
	SubTemplate[U]
	baseData *T
}

// Base data used to define the data passed in to the
// template
type BaseData[T any, U any] struct {
	B T // BaseData passed in on every Render() call
	D U // Data passed in explicitly by the Render(data) call
}

func NewTemplate[T, U any](tc *TemplateContext[T], data U) *Template[T, U] {
	t := Template[T, U]{
		SubTemplate: SubTemplate[U]{
			ctx:  tc.templateContextCore,
			data: data,
		},
		baseData: tc.baseData,
	}

	registry.Add(&t)

	return &t
}

var ErrNoBasePatternFound = errors.New("no basetemplate has been provided, but NewTemplate was called")

// Loads, validates and registers the template.
// This should rarely be called directly
func (t *Template[T, U]) Load() error {

	if len(t.ctx.baseTemplates) == 0 {
		return TemplateError{t.ctx, t.usePattern, ErrNoBasePatternFound}
	}
	t.usePattern = filepath.Base(t.ctx.baseTemplates[0])

	err := t.load(BaseData[T, U]{B: *t.baseData, D: t.data})
	if errors.Is(err, ErrTemplateExecute) {
		return fmt.Errorf("%w: has .B or .D prefix been included for this Template?", err)
	}
	return err
}

// Renders the template to a writer with the base data
// and data of the loaded type.
// The data injected into a struct is of the form:
//
//	{
//			B: any // Base data
//			D: any // Data as passed in through the Render
//	}
//
// Even if no base data has been provided, the template will be provided
// in the above form. If live reloading is enabled, JS is injected at the end of the body.
//
// If handling io.Writer errors is required, it is suggested to wrap the io.Writer
// in a custom writer that returns an error, for example:
//
//	type wrapWriter struct {
//		w   io.Writer
//		err error
//	}
//
//	func (w *wrapWriter) Write(p []byte) (int, error) {
//		if w.err != nil {
//			return 0, w.err
//		}
//
//		n, err := w.w.Write(p)
//		if err != nil {
//			w.err = err
//		}
//
//		return n, err
//		}
func (t *Template[T, U]) Render(w io.Writer, data U) {
	d := BaseData[T, U]{B: *t.baseData, D: data}
	t.render(w, d)
}

type SubTemplate[U any] struct {
	t          *template.Template
	ctx        templateContextCore
	usePattern string
	data       U
}

func NewSubTemplate[T, U any](tc *TemplateContext[T], pattern string, data U) *SubTemplate[U] {
	t := SubTemplate[U]{
		ctx:        tc.templateContextCore,
		data:       data,
		usePattern: pattern,
	}

	registry.Add(&t)

	return &t
}

func (t *SubTemplate[U]) Load() error {
	return t.load(t.data)
}

func (t *SubTemplate[U]) Render(w io.Writer, data U) {
	t.render(w, data)
}

var ErrNoConfigProvided = errors.New("no config provided")
var ErrNoBaseOrPatternFound = errors.New("no basetemplate nor patterns have been provided")
var ErrTemplateParse = errors.New("template parse error")

func (t *SubTemplate[U]) load(data any) error {
	// Immeditately run on load
	if t.ctx.onLoad != nil {
		err := t.ctx.onLoad()
		if err != nil {
			return err
		}
	}

	if t.ctx.config == nil {
		return ErrNoConfigProvided
	}

	patterns := []string{}
	patterns = append(patterns, t.ctx.baseTemplates...)
	patterns = append(patterns, t.ctx.withTemplates...)

	if len(patterns) == 0 {
		return TemplateError{t.ctx, "", ErrNoBaseOrPatternFound}
	}

	// Parse and cache the template
	var err error
	t.t, err = template.New("").Funcs(*t.ctx.funcMap).ParseFS(t.ctx.config.FS, patterns...)
	if err != nil {
		return TemplateError{t.ctx, t.usePattern, fmt.Errorf("%w: %v", ErrTemplateParse, err)}
	}

	var buf bytes.Buffer
	err = t.t.ExecuteTemplate(&buf, t.usePattern, data)
	if err != nil {
		return TemplateError{t.ctx, t.usePattern, fmt.Errorf("%w: %v", ErrTemplateExecute, err)}
	}

	return nil

}

type failWriter struct {
	w   io.Writer
	err error
}

// failWriter is a custom io.Writer that captures the first error
// that occurs during writing. This is necessary to discern between
// template rendering errors and writer errors due to how
// template.ExecuteTemplate works.
func (fw *failWriter) Write(p []byte) (int, error) {
	if fw.err != nil {
		return 0, fw.err
	}

	n, err := fw.w.Write(p)
	switch err {
	case http.ErrBodyNotAllowed, http.ErrHijacked, http.ErrContentLength:
		// these are edgecase implementation bugs on the server, panic to notify implementation
		fw.err = err
	case nil:
		fw.err = nil
	default:
		// Any other error is likely a client disconnect, ignore it
		fw.err = nil
	}

	return n, fw.err
}

// render is the actual implementation to render the template.
func (t *SubTemplate[U]) render(w io.Writer, d any) {

	fw := &failWriter{w: w}

	// Without reload, rendering is short and simple
	if !registry.LiveReload() {
		err := t.t.ExecuteTemplate(fw, t.usePattern, d)
		if err != nil {
			// Panics if the template fails to be written due to a server error
			panic(&TemplateError{t.ctx, t.usePattern, fmt.Errorf("%w %s", ErrTemplateExecute, err)})
		}

		return
	}

	// Reload the component
	err := t.load(d)
	if err != nil {
		livereload.Notify(err)

		// To allow for SSE to work even if the template fails to load,
		// the bare JS must be injected to allow for reconnection
		_, err := fw.Write([]byte(registry.JSToInject()))
		if err != nil {
			panic(&TemplateError{t.ctx, t.usePattern, fmt.Errorf("%w %s", ErrTemplateExecute, err)})
		}

		return
	}

	var buf bytes.Buffer
	// Capture the output to a buffer to inject the necessary JS
	err = t.t.ExecuteTemplate(&buf, t.usePattern, d)
	if err != nil {
		panic(&TemplateError{t.ctx, t.usePattern, fmt.Errorf("%w %s", ErrTemplateExecute, err)})
	}

	html := buf.String()
	idx := strings.LastIndex(strings.ToLower(html), "</body>")
	if idx != -1 {
		html = html[:idx] + registry.JSToInject() + html[idx:]
	}

	_, err = fw.Write([]byte(html))
	if err != nil {
		panic(&TemplateError{t.ctx, t.usePattern, fmt.Errorf("%w %s", ErrTemplateExecute, err)})
	}
}
