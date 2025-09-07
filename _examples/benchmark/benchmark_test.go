package loadr

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/nesbyte/loadr"
)

// Please note, this is a micro benchmark
var htmlDir = os.DirFS(".")

var config = loadr.BaseConfig{
	FS: htmlDir}

var base = loadr.NewTemplateContext(config, loadr.NoData, "index.html", "components.html")

type testData struct {
	Test string
}

var sample = testData{Test: "Hello World!"}
var sampleSizes = []int{1e0, 1e3, 1e6}

// Using html/templates caching the parsed template
func BenchmarkStdTemplates(b *testing.B) {
	t, err := template.ParseFS(htmlDir, "index.html", "components.html")
	if err != nil {
		b.Fatal(err)
	}

	for _, size := range sampleSizes {
		data := testData{}
		data.Test = strings.Repeat(sample.Test, size)

		b.Run(fmt.Sprintf(
			"Size_%d", size),
			func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					var bs bytes.Buffer
					bs.Reset()
					err := t.ExecuteTemplate(&bs, "index.html", struct{ D testData }{data})
					if err != nil {
						b.Fatal(err)
					}
				}
			})
	}
}

// Using loadr with templates loaded
func BenchmarkLoadrInProductionMode(b *testing.B) {

	t := loadr.NewTemplate(base, testData{})
	err := loadr.LoadTemplates()
	if err != nil {
		b.Fatal(err)
	}

	for _, size := range sampleSizes {
		data := testData{}
		data.Test = strings.Repeat(sample.Test, size)

		b.Run(fmt.Sprintf(
			"Size_%d", size),
			func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					var bs bytes.Buffer
					bs.Reset()
					t.Render(&bs, data)
				}
			})
	}
}

// Using html/templates with the templates re-parsed on every iteration
func BenchmarkStdTemplatesWithLiveReload(b *testing.B) {

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
		loadr.MustRunLiveReload("/event", nil, ".")
	})

	t := loadr.NewTemplate(base, testData{})
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
