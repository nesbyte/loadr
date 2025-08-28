package core

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/nesbyte/loadr/livereload"
	"github.com/nesbyte/loadr/registry"
)

func NewTemplate[T, U any](tc *TemplateContext[T], data U) *Template[T, U] {
	t := Template[T, U]{
		templateCore: templateCore[U]{
			ctx:  &tc.templateContextCore,
			data: data,
		},
		baseData: tc.baseData,
	}

	registry.Add(&t)

	return &t
}

func NewSubTemplate[T, U any](tc *TemplateContext[T], pattern string, data U) *SubTemplate[U] {
	t := SubTemplate[U]{
		templateCore: templateCore[U]{
			ctx:        &tc.templateContextCore,
			data:       data,
			usePattern: pattern,
		},
	}

	registry.Add(&t)

	return &t
}

type templateCore[U any] struct {
	ctx        *templateContextCore
	t          *template.Template
	usePattern string
	data       U
}

type SubTemplate[U any] struct {
	templateCore[U]
}

func (t *SubTemplate[U]) Render(w io.Writer, data U) {
	err := t.render(w, data)
	if err != nil {
		panic(LoadingError{t.ctx, t.usePattern, fmt.Errorf("execute template error in render %s", err)})
	}
}

func (t *SubTemplate[U]) Load() error {
	return t.load(t.data)
}

type Template[T, U any] struct {
	templateCore[U]
	baseData *T
}

// Base data used to define the data passed in to the
// template
type BaseData[T any, U any] struct {
	B T // BaseData passed in on every Render() call
	D U // Data passed in explicitly by the Render(data) call
}

func (t *Template[T, U]) Load() error {
	if len(t.ctx.baseTemplates) == 0 {
		return ErrNoBasePatternFound
	}

	t.usePattern = t.ctx.baseTemplates[0]
	d := &BaseData[T, U]{B: *t.baseData, D: t.data}
	return t.load(d)
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

	t.usePattern = t.ctx.baseTemplates[0]

	d := &BaseData[T, U]{B: *t.baseData, D: data}

	err := t.render(w, d)
	if err != nil {
		log.Println(err)
		panic(LoadingError{t.ctx, t.usePattern, fmt.Errorf("execute template error in render %s", err)})
	}
}

var ErrNoBaseOrPatternFound = errors.New("no basetemplate nor patterns have been provided")
var ErrNoBasePatternFound = errors.New("no basetemplate has been provided, but NewTemplate was called")

type LoadingError struct {
	tc         *templateContextCore
	UsePattern string
	Err        error
}

func (e LoadingError) Error() string {
	return fmt.Sprintf("basetemplates %q with templates %q and template pattern %q failed: %s", e.tc.baseTemplates, strings.Join(e.tc.withTemplates, ", "), e.UsePattern, e.Err.Error())
}

func (e LoadingError) Unwrap() error {
	return e.Err
}

var ErrNoConfigProvided = errors.New("no config provided")
var ErrTemplateParse = errors.New("template parse error")
var ErrInvalidTemplateData = errors.New("invalid template data")

// Loads, validates and registers the template.
func (tc *templateCore[U]) load(data any) error {
	// Immeditately run on load
	if tc.ctx.onLoad != nil {
		err := tc.ctx.onLoad()
		if err != nil {
			return err
		}
	}

	if tc.ctx.config == nil {
		return ErrNoConfigProvided
	}

	patterns := []string{}
	patterns = append(patterns, tc.ctx.baseTemplates...)
	patterns = append(patterns, tc.ctx.withTemplates...)

	if len(patterns) == 0 {
		return LoadingError{tc.ctx, tc.usePattern, ErrNoBaseOrPatternFound}
	}

	// Parse and cache the template
	var err error
	tc.t, err = template.New("").Funcs(tc.ctx.funcMap).ParseFS(tc.ctx.config.FS, patterns...)
	if err != nil {
		return LoadingError{tc.ctx, tc.usePattern, fmt.Errorf("%w: %v", ErrTemplateParse, err)}
	}

	bs := []byte{}
	w := bytes.NewBuffer(bs)
	err = tc.t.ExecuteTemplate(w, tc.usePattern, data)
	if err != nil {
		return LoadingError{
			tc.ctx,
			tc.usePattern,
			fmt.Errorf("%w has a .B or .D prefix been included for the field?: %v", ErrInvalidTemplateData, err),
		}
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
	if err != nil {
		fw.err = err
	}

	return n, err
}

// render is the actual implementation to render the template.
func (tc templateCore[U]) render(w io.Writer, data any) error {

	fw := &failWriter{w: w}

	// Without reload, rendering is short and simple
	if !registry.LiveReload() {
		err := tc.t.ExecuteTemplate(fw, tc.usePattern, data)
		if fw.err != nil {
			switch fw.err {
			case http.ErrBodyNotAllowed, http.ErrHijacked, http.ErrContentLength:
				return fw.err
			default:
			}
			// Ignore any other error, such as io.Writer errors
			return nil
		}
		if err != nil {
			// This should never happen as the template has been validated and should be handeled as a panic
			return err
		}
		return nil
	}

	// Reload the component
	err := tc.load(data)
	if err != nil {
		livereload.LiveReloadCustomErrorHandler(err)
		_, err := fw.Write([]byte(registry.JSToInject()))
		return err
	}

	// Capture the output to a buffer to inject the necessary JS
	var buf bytes.Buffer
	err = tc.t.ExecuteTemplate(&buf, tc.usePattern, data)
	if err != nil {
		// This should never happen as the template has been validated and should be handeled as a panic
		return fmt.Errorf("execute template error in render %s", err)
	}

	html := buf.String()
	idx := strings.LastIndex(strings.ToLower(html), "</body>")
	if idx != -1 {
		html = html[:idx] + registry.JSToInject() + html[idx:]
	}

	_, err = w.Write([]byte(html))

	return err
}
