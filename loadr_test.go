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

type case1BaseData struct {
	Title string
}

type case1Partial1 struct {
	Sample string
}

type case1Partial2 struct {
	Sample2 string
}

type testScenario struct {
	name      string
	input     any
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

	table := []testScenario{
		{"get input.html with partial1",
			NewTemplate(p1, "input.html",
				case1Partial1{}),
			"want.input1.html",
			nil},
		{"get partial as partial1.html, should return empty",
			NewTemplate(p1, "input.partial1.html", case1Partial1{}),
			"want.empty.html",
			nil},
		{"get partial as partial1",
			NewTemplate(p1, "partial", case1Partial1{}),
			"want.partial1.html",
			nil},
		{"get partial as partial1 with wrong data format",
			NewTemplate(p1, "partial", case1Partial2{}),
			"",
			core.ErrInvalidTemplateData},
		{"get input.html with partial2",
			NewTemplate(p2, "input.html", case1Partial2{}),
			"want.input2.html",
			nil},
		{"get partial as partial2",
			NewTemplate(p2, "partial", case1Partial2{}),
			"want.partial2.html",
			nil},
		{"get input.html with partial3",
			NewTemplate(p3, "partial", []string{}),
			"want.partial3.html",
			nil},
	}

	// Runs the table test
	bs := []byte{}
	wr := bytes.NewBuffer(bs)
	for _, scenario := range table {

		wr.Reset()

		switch v := scenario.input.(type) {
		case *core.Templ[case1BaseData, case1Partial1]:
			err := v.Load()
			if !scenario.ShouldRender(t, err) {
				continue
			}
			v.Render(wr, case1Partial1{}) // renders
		case *core.Templ[case1BaseData, case1Partial2]:
			err := v.Load()
			if !scenario.ShouldRender(t, err) {
				continue
			}
			v.Render(wr, case1Partial2{})
		case *core.Templ[case1BaseData, []string]:
			err := v.Load()
			if !scenario.ShouldRender(t, err) {
				continue
			}
			v.Render(wr, []string{})
		}
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

	_ = NewTemplate(b, "input.partial1.html", case1Partial1{})
	_ = NewTemplate(cp, "input.partial2.html", case1Partial2{})

	err := LoadTemplates()
	if err != nil {
		t.Error(err)
	}

	// And prove that there is no input.partial1.html
	_ = NewTemplate(cp, "input.partial1.html", case1Partial2{})
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
	templ := NewTemplate(b, "input.emptydata.html", NoData)

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
	_, cancel, err := RunLiveReload("/live-reload", HandleReload, "testdata")
	if err != nil {
		t.Error(err)
	}
	defer cancel()

	_, cancel, err = RunLiveReload("/live-reload2", HandleReload, "testdata")
	if err == nil {
		t.Error("want error, live reload cannot be called twice")
		defer cancel()
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

	index := NewTemplate(base, "input.html", upperData{"test"})

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
