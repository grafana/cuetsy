package cuetsy_test

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"github.com/grafana/cuetsy"
	"github.com/grafana/cuetsy/internal/cuetxtar"
)

func TestGenerateWithImports(t *testing.T) {
	test := cuetxtar.TxTarTest{
		Root: "./testdata/imports",
		Name: "gen",
		ToDo: map[string]string{
			"imports/oneref_verbose":   "Figure out how to disambiguate struct literals from the struct-with-braces-and-one-element case",
			"imports/struct_shorthand": "Shorthand struct notation is currently unsupported, needs fixing",
		},
	}

	ctx := cuecontext.New()

	test.Run(t, func(t *cuetxtar.Test) {
		v := ctx.BuildInstance(t.ValidInstances()[0])
		if v.Err() != nil {
			t.Fatal(v.Err())
		}

		b, err := cuetsy.Generate(v, cuetsy.Config{})
		if err != nil {
			errors.Print(t, err, nil)
			t.Fatal(errors.Details(err, nil))
		}

		_, _ = t.Write(b)
	})
}
