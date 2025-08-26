package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/nesbyte/loadr"
)

//go:embed "*"
var baseFS embed.FS

// By default (production), use the embedded file system
var config = loadr.BaseConfig{
	FS: baseFS,
}

type baseData struct {
	Styles string
	JSDist string
}

// Make package global templates. In larger projects
// the templates can then be parsed once and imported everywhere.

// Creates the initial template and sets an initial config and the
// expected base data type
// Base data is the data which will be made available in *all* templates
// derived from this base template
var base = loadr.NewTemplateContext(
	config,
	baseData{},
	"index.html",
	"global_components.html")

// The defined data that the index render function takes in
type IndexData struct {
	Name    string
	Content string
}

// This extracts the template of interest from base, and provides
// The template specific data type
var index = loadr.NewTemplate(base, IndexData{})

// Some data for the content template
type ContentData struct {
	Content string
}

// Extracts another template of interest with it's specific data type
var content = loadr.NewSubTemplate(base, "content", ContentData{})

// Bringing it all together below
func main() {

	base.SetBaseData(baseData{"styles.[somehash].css", "dist/bundle.[somehash].js"})
	base.Funcs(template.FuncMap{
		"customFunc": func(s string) string {
			return fmt.Sprintf("Applying the funcmap: %s!!", s)
		},
	})

	err := loadr.LoadTemplates()
	if err != nil {
		log.Fatalln(err)
	}

	r := http.NewServeMux()
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		index.Render(w, IndexData{"Alice", "Index Injected Content"})
	})

	r.HandleFunc("/content", func(w http.ResponseWriter, r *http.Request) {
		content.Render(w, ContentData{"Specific Template content"})
	})

	fmt.Println("Listening on 8080, open http://localhost:8080/")
	err = http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatalln(err)
	}
}
