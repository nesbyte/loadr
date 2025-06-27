package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"

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
var base = loadr.NewBaseTemplate(baseData{}).SetConfig(config).SetBaseTemplates(
	"index.html",
	"global_components.html")

// The defined data that the index render function takes in
type IndexData struct {
	Name    string
	Content string
}

// This extracts the template of interest from base, and provides
// The template specific data type
var index = loadr.NewTemplate(base, "index.html", IndexData{})

// Some data for the content template
type ContentData struct {
	Content string
}

// Extracts another template of interest with it's specific data type
var content = loadr.NewTemplate(base, "content", ContentData{})

// Bringing it all together below
func main() {

	// This can also be set outside of a function however, it is primarely
	// for making things such as cache busting easier and provide
	// essentially constant data.
	// If SetBaseData is called again, the update will propagate to all child templates
	// immediately on next Render() call regardless of the LoadTemplates call.
	base.SetBaseData(baseData{"styles.[somehash].css", "dist/bundle.[somehash].js"})

	// Let's create a basic web server
	r := http.NewServeMux()

	// When in dev mode, live-reloading can be enabled
	// If you run the server feel free to change the index.html, save and see the page
	// update on the fly without needing to recompile!
	// Breaking the fields provide verbose error messages
	liveReload := false
	if liveReload {

		// In dev, use the os filesystem instead of the embedded one
		// to allow for live reloads to take place
		var config = loadr.BaseConfig{
			FS: os.DirFS("."),
		}
		base.SetConfig(config) // resetting the config is ok!

		// Live reload takes in the pattern of which the HTTP server will listen on (/live-reload)
		// and allows some insertion of custom logic of what to do if a file has changed.
		// HandleReload is a default setup that simply prints out reloaded files or errors
		// pathsToWatch will recursively watch all files and folders inside itself
		lsHandler, lsClose, err := loadr.RunLiveReload("/live-reload", loadr.HandleReload, ".")
		if err != nil {
			log.Fatalln(err)
		}
		defer lsClose() // Not strictly necessary, but cleans things up inside the live reloader

		// use the handler provided by the LiveReload function.
		r.Handle("/live-reload", lsHandler)

	}

	// This should be called after all loadr interactions are set
	// If this is not run last any changes may not propagate properly
	// to the templates.
	err := loadr.LoadTemplates()
	if err != nil {
		log.Fatalln(err)
	}

	// The rendering is called in here
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Notice how Render() does not return an error?
		// the error handling is done and returned by LoadTemplates()
		// earlier on.
		index.Render(w, IndexData{"Alice", "Index Injected Content"})
	})

	r.HandleFunc("/content", func(w http.ResponseWriter, r *http.Request) {
		content.Render(w, ContentData{"Specific Template content"})
	})

	// Uncomment the below and see that it will not compile as the content template is expecting
	// a ContentData type. Type safety out of the box!
	// r.HandleFunc("/breaks", func(w http.ResponseWriter, r *http.Request) {
	// 	content.Render(w, IndexData{"Breaks!", "breaks"})
	// })

	fmt.Println("Listening on 8080, open http://localhost:8080/")
	err = http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatalln(err)
	}
}
