package encoder_test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/pkg/strings"
	"github.com/google/go-cmp/cmp"
	"github.com/sdboyer/cuetsy/encoder"
	"golang.org/x/tools/txtar"
	"gotest.tools/assert"
)

const CasesDir = "tests"

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
			out, err := encoder.Generate(i.Value(), encoder.Config{})
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
		file := filepath.Join(dir, fi.Name())
		a, err := txtar.ParseFile(file)
		if err != nil {
			return nil, err
		}

		if len(a.Files) != 2 {
			return nil, fmt.Errorf("Malformed test case '%s': Must contain exactly two files (CUE and TS/ERR), but has %d", file, len(a.Files))
		}
		if strings.HasSuffix(strings.TrimSuffix(fi.Name(), ".txtar"), "error") {
			cases = append(cases, Case{
				CaseType: ErrorType,
				Name:     fi.Name(),
				CUE:      string(a.Files[0].Data),
				ERROR:    strings.TrimSuffix(string(a.Files[1].Data), "\n"),
			})
		} else {
			cases = append(cases, Case{
				CaseType: TSType,
				Name:     fi.Name(),
				CUE:      string(a.Files[0].Data),
				TS:       string(a.Files[1].Data),
			})
		}
	}
	return cases, nil
}
