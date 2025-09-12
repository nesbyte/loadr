package main

import (
	"embed"
	"fmt"
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

// Notice the WithTemplates method, this allows for composition of templates
// WT is just an alias for WithTemplates
// It is important that the base template is still used, the standard templates/html
// functionality will automatically parse the relevant templates under the hood as long as they are
// passed in WithTemplates
// WithTemplates does not mutate the base template, it returns a new one
var indexWithComposition1 = loadr.NewTemplate(base.WithTemplates("composition1/*.html"), IndexData{})
var indexWithComposition2 = loadr.NewTemplate(base.WT("composition2/*.html"), IndexData{})

// Again highlighting that the base template is not mutated when with templates is used
var index = loadr.NewTemplate(base, loadr.NoData)

// Bringing it all together below
func main() {

	base.SetBaseData(baseData{"styles.[somehash].css", "dist/bundle.[somehash].js"})

	// Let's create a basic web server
	r := http.NewServeMux()

	// This should be called after all loadr interactions are set
	// If this is not run last any changes may not propagate properly
	// to the templates.
	err := loadr.LoadTemplates()
	if err != nil {
		log.Fatalln(err)
	}

	// BELOW IS THE COMPARISON BETWEEN PREVIOUS AND PROPOSED BEHAVIOUR

	// r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	index.Render(w, loadr.NoData)
	// })

	r.HandleFunc("/", index.HandlerFunc())

	r.HandleFunc("/composition1", func(w http.ResponseWriter, r *http.Request) {
		// Notice that using the composition does not really incur any extra
		// complexity
		indexWithComposition1.Render(w, IndexData{"Alice", "Composition 1 Injected Content"})
	})

	// The rendering is called in here
	r.HandleFunc("/composition2", func(w http.ResponseWriter, r *http.Request) {
		indexWithComposition2.Render(w, IndexData{"Bob", "Composition 2 Injected Content"})
	})

	fmt.Println("Listening on 8080, open http://localhost:8080/")
	err = http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatalln(err)
	}
}
