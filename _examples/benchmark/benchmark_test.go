package loadr

import (
	"bytes"
	"html/template"
	"os"
	"sync"
	"testing"

	"github.com/nesbyte/loadr"
)

// Please note, this is a micro benchmark
var htmlDir = os.DirFS(".")

var config = loadr.BaseConfig{
	FS: htmlDir}

var base = loadr.NewBaseTemplate("").SetConfig(config).SetBaseTemplates("index.html", "components.html")

type testData struct {
	Test string
}

var sample = testData{Test: "Hello World!"}

// Using html/templates caching the parsed template
func BenchmarkStdTemplates(b *testing.B) {
	t, err := template.ParseFS(htmlDir, "index.html", "components.html")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var bs bytes.Buffer
		bs.Reset()
		t.ExecuteTemplate(&bs, "index.html", sample)
	}
}

// Using loadr with templates loaded
func BenchmarkLoadrInProductionMode(b *testing.B) {

	t := loadr.NewTemplate(base, "index.html", testData{})
	err := loadr.LoadTemplates()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var bs bytes.Buffer
		bs.Reset()
		t.Render(&bs, sample)
	}
}

// Using html/templates with the templates re-parsed on every iteration
func BenchmarkStdTemplatesWithParsing(b *testing.B) {

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t, err := template.ParseFS(htmlDir, "index.html", "components.html")
		if err != nil {
			b.Fatal(err)
		}
		var bs bytes.Buffer
		bs.Reset()
		t.ExecuteTemplate(&bs, "index.html", sample)
	}
}

var once sync.Once

// Using loadr with live reload enabled
func BenchmarkLoadrWithLiveReload(b *testing.B) {
	once.Do(func() {
		_, _, err := loadr.RunLiveReload("/event", loadr.HandleReloadError, ".")
		if err != nil {
			b.Fatal(err)
		}

	})

	t := loadr.NewTemplate(base, "index.html", testData{})
	err := loadr.LoadTemplates()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var bs bytes.Buffer
		bs.Reset()
		t.Render(&bs, sample)
	}
}
