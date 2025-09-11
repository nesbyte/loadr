package loadr

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/nesbyte/loadr/core"
	"github.com/nesbyte/loadr/registry"
)

const case1Dir = "./testdata/case1"
const case2Dir = "./testdata/case2"
const case3Dir = "./testdata/case3"
const case4Dir = "./testdata/case4"

type case1BaseData struct {
	Title string
}

type case1Partial1 struct {
	Sample string
}

type case1Partial2 struct {
	Sample2 string
}

type Renderable interface {
	Load() error
	Render(w io.Writer, data any)
}

type testScenario struct {
	name      string
	input     Renderable
	data      any
	wantId    string
	wantError error
}

func (s testScenario) ShouldRender(t *testing.T, err error) bool {
	if err == s.wantError {
		return true
	}

	if errors.Is(err, s.wantError) {
		return false
	} else {
		t.Errorf("Scenario: %s\nwant error: %s\ngot error: %s\n", s.name, s.wantError, err)
		return false
	}
}

func TestCase1PartialsFromWithTemplates(t *testing.T) {
	var (
		caseFS = os.DirFS(case1Dir)
	)

	b := NewTemplateContext(
		BaseConfig{FS: caseFS},
		case1BaseData{},
		"input.html",
	)

	p1 := b.WithTemplates("input.partial1.html")
	p2 := b.WithTemplates("input.partial2.html")
	p3 := b.WithTemplates("input.partial3.html")
	defer registry.Reset()

	// any is used in the New*Template() to avoid type issues in the table test
	// since the data types are different
	table := []testScenario{
		{"get input.html with partial1",
			NewTemplate(p1,
				any(case1Partial1{Sample: ""})),
			case1Partial1{""},
			"want.input1.html",
			nil},
		{"get partial as partial1.html, should return empty",
			NewSubTemplate(p1, "input.partial1.html", any(case1Partial1{})),
			case1Partial1{""},
			"want.empty.html",
			nil},
		{"get partial as partial1",
			NewSubTemplate(p1, "partial", any(case1Partial1{})),
			case1Partial1{},
			"want.partial1.html",
			nil},
		{"get partial as partial1 with wrong data format",
			NewSubTemplate(p1, "partial", any(case1Partial2{})),
			case1Partial2{},
			"",
			core.ErrTemplateExecute},
		{"get input.html with partial2",
			NewTemplate(p2, any(case1Partial2{})),
			case1Partial2{},
			"want.input2.html",
			nil},
		{"get partial as partial2",
			NewSubTemplate(p2, "partial", any(case1Partial2{})),
			case1Partial2{},
			"want.partial2.html",
			nil},
		{"get input.html with partial3",
			NewSubTemplate(p3, "partial", any([]string{})),
			[]string{},
			"want.partial3.html",
			nil},
	}

	// Runs the table test
	bs := []byte{}
	wr := bytes.NewBuffer(bs)
	for _, scenario := range table {

		wr.Reset()

		err := scenario.input.Load()
		if !scenario.ShouldRender(t, err) {
			continue
		}
		scenario.input.Render(wr, scenario.data) // renders

		testContent := wr.String()

		// Gets or creates the golden file
		f, err := caseFS.Open(scenario.wantId)
		if err != nil {
			// If file does not exist create a golden file
			// the test will still error out
			err = os.WriteFile(fmt.Sprintf("%s/%s", case1Dir, scenario.wantId), []byte(testContent), 0644)
			if err != nil {
				t.Fatal(err)
			}
			t.Fatalf("GOLDEN FILE: %s CREATED, TEST WILL FAIL FIRST TIME", scenario.wantId)
		}
		bGolden, err := io.ReadAll(f)
		if err != nil {
			f.Close()
			t.Fatal(err)
		}
		f.Close()

		if strings.TrimSpace(testContent) != strings.TrimSpace(string(bGolden)) {
			t.Errorf("Scenario: %s\n\nwant:\n%s\n\ngot:\n%s\n", scenario.name, string(bGolden), wr.String())
		}

	}
}

var errorSimulatedWrite = errors.New("simulated write error")

type alwaysFailWriter struct{} // dummy writer that always fails

func (fw *alwaysFailWriter) Write(p []byte) (int, error) {
	return 0, errorSimulatedWrite
}

// Shows how to wrap a writer to capture the error
type wrapWriter struct {
	w   io.Writer
	err error
}

func (w *wrapWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}

	n, err := w.w.Write(p)
	if err != nil {
		w.err = err
	}

	return n, err
}

// Validates that RenderWithError captures write errors correctly
func TestRenderWithWriterError(t *testing.T) {
	var (
		caseFS = os.DirFS(case1Dir)
	)

	b := NewTemplateContext(
		BaseConfig{FS: caseFS},
		case1BaseData{},
		"input.html",
		"input.partial1.html",
	)

	p1 := NewTemplate(b, case1Partial1{})

	err := LoadTemplates()
	if err != nil {
		t.Error(err)
	}

	// Makes sure render does not panic on write errors
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("unexpected panic: %v", r)
		}
	}()

	// Even though the writer fails, it should not panic
	p1.Render(&alwaysFailWriter{}, case1Partial1{"test"})

	// Wrapping the writer, it should be possible to extract the error
	// and it should not panic
	wrappedWriter := &wrapWriter{w: &alwaysFailWriter{}}
	p1.Render(wrappedWriter, case1Partial1{"test"})
	if wrappedWriter.err != nil {
	} else {
		t.Error("expected error, wrappedWriter should have an error")
	}
}

func TestBaseCopy(t *testing.T) {
	var (
		caseFS = os.DirFS(case1Dir)
	)

	b := NewTemplateContext(
		BaseConfig{FS: caseFS},
		case1BaseData{},
		"input.partial1.html",
	)

	defer registry.Reset()

	cp := b.Copy()
	cp.SetBaseTemplates("input.partial2.html")

	_ = NewTemplate(b, case1Partial1{})
	_ = NewTemplate(cp, case1Partial2{})

	err := LoadTemplates()
	if err != nil {
		t.Error(err)
	}

	// And prove that there is no input.partial1.html
	_ = NewSubTemplate(cp, "input.partial1.html", case1Partial2{})
	err = LoadTemplates()
	if err == nil {
		t.Error("expected error, input.partial1.html should not exist")
	}
}

// Ensure that changes in base data propagate
// immediately
func TestBaseDataImmediatePropagation(t *testing.T) {
	var (
		caseFS = os.DirFS(case2Dir)
	)

	type caseData struct {
		Title int
	}

	defer registry.Reset()
	b := NewTemplateContext(BaseConfig{FS: caseFS}, caseData{1}, "input.emptydata.html")
	templ := NewTemplate(b, NoData)

	err := LoadTemplates()
	if err != nil {
		t.Error(err)
	}

	bs := []byte{}
	wr := bytes.NewBuffer(bs)
	for i := 0; i < 5; i++ {
		wr.Reset()
		b.SetBaseData(caseData{i})
		templ.Render(wr, NoData)

		rs := wr.String()
		if rs != strconv.Itoa(i) {
			t.Errorf("\nwant: %d\ngot: %s\n", i, rs)
		}
	}
}

func TestLiveReloadCallTwice(t *testing.T) {
	_, err := RunLiveReload("/live-reload", nil, "testdata")
	if err != nil {
		t.Error(err)
	}

	_, err = RunLiveReload("/live-reload2", nil, "testdata")
	if err == nil {
		t.Error("want error, live reload cannot be called twice")
	}

}

// Validates that the FuncMap functionality works as expected
func TestFuncMapFunctionality(t *testing.T) {
	var (
		caseFS = os.DirFS(case3Dir)
	)

	funcMap := template.FuncMap{
		"toUpper": strings.ToUpper,
	}

	type upperData struct {
		Name string
	}

	base := NewTemplateContext(
		BaseConfig{FS: caseFS},
		NoData,
		"input.html",
	).Funcs(funcMap)

	index := NewTemplate(base, upperData{"test"})

	err := LoadTemplates()
	if err != nil {
		t.Fatalf("loadtemplates failed: %s", err)
	}

	b := bytes.NewBufferString("")
	index.Render(b, upperData{"test"})

	if b.String() != "TEST" {
		t.Errorf("want: TEST\ngot: %s\n", b.String())
	}
}

// Validates that nested base templates work as expected
// this is a regression test as the html/template package
// only uses the last element of a path as the template name
func TestNestedBaseTempalte(t *testing.T) {
	var (
		caseFS = os.DirFS(case4Dir)
	)

	base := NewTemplateContext(
		BaseConfig{FS: caseFS},
		NoData,
		"folder/index.html",
	)
	_ = NewTemplate(base, NoData)

	err := LoadTemplates()
	if err != nil {
		t.Fatalf("loadtemplates failed: %s", err)
	}
}
