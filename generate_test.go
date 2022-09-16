package cuetsy_test

import (
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/pkg/strings"
	"github.com/google/go-cmp/cmp"
	"github.com/grafana/cuetsy"
	"github.com/grafana/cuetsy/internal/cuetxtar"
	"golang.org/x/tools/txtar"
	"gotest.tools/assert"
)

const CasesDir = "testdata"

type TestCaseType int

const (
	TSType    TestCaseType = 0
	ErrorType TestCaseType = 1
)

type Case struct {
	CaseType TestCaseType
	Name     string

	CUE   string
	TS    string
	ERROR string
}

var updateGolden = flag.Bool("update-golden", false, "Update golden files with test results")

func TestGenerateWithImports(t *testing.T) {
	test := cuetxtar.TxTarTest{
		Root:   "./testdata/imports",
		Name:   "gen",
		Update: *updateGolden,
		ToDo: map[string]string{
			"imports/oneref_verbose":   "Figure out how to disambiguate struct literals from the struct-with-braces-and-one-element case",
			"imports/struct_shorthand": "Shorthand struct notation is currently unsupported, needs fixing",
			"imports/single_embed":     "Single-item struct embeds should be treated as just another interface to compose, but get confused with references - #60",
		},
	}

	ctx := cuecontext.New()

	test.Run(t, func(t *cuetxtar.Test) {
		v := ctx.BuildInstance(t.ValidInstances()[0])
		if v.Err() != nil {
			t.Fatal(v.Err())
		}

		b, err := cuetsy.Generate(v, cuetsy.Config{
			Export: true,
		})
		if err != nil {
			errors.Print(t, err, nil)
			t.Fatal(errors.Details(err, nil))
		}

		_, _ = t.Write(b)
	})
}

func TestGenerate(t *testing.T) {
	cases, err := loadCases(CasesDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ctx := cuecontext.New()
			i := ctx.CompileString(c.CUE, cue.Filename(c.Name+".cue"))
			if err != nil {
				t.Fatal(err)
			}
			out, err := cuetsy.Generate(i.Value(), cuetsy.Config{
				Export: true,
			})
			if c.CaseType == ErrorType {
				assert.Error(t, err, c.ERROR)
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if s := cmp.Diff(c.TS, string(out)); s != "" {
					t.Fatal(s)
				}
			}
		})
	}
}

func loadCases(dir string) ([]Case, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var cases []Case

	for _, fi := range files {
		if fi.IsDir() {
			continue
		}
		file := filepath.Join(dir, fi.Name())
		a, err := txtar.ParseFile(file)
		if err != nil {
			return nil, err
		}

		if len(a.Files) != 2 {
			return nil, fmt.Errorf("malformed test case '%s': Must contain exactly two files (CUE and TS/ERR), but has %d", file, len(a.Files))
		}
		name := strings.TrimSuffix(fi.Name(), ".txtar")
		if strings.HasSuffix(name, "error") {
			cases = append(cases, Case{
				CaseType: ErrorType,
				Name:     name,
				CUE:      string(a.Files[0].Data),
				ERROR:    strings.TrimSuffix(string(a.Files[1].Data), "\n"),
			})
		} else {
			cases = append(cases, Case{
				CaseType: TSType,
				Name:     name,
				CUE:      string(a.Files[0].Data),
				TS:       string(a.Files[1].Data),
			})
		}
	}
	return cases, nil
}
