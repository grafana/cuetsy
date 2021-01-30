package encoder

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"sort"
	"strings"
	"text/template"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/errors"
)

// TODO use txtar to set up a buncha test cases

// Generate takes a cue.Instance and generates the corresponding Typescript.
//
// It is expected that the cue.Instance represents a top-level struct - that is,
// the contents of a single file or merged contents of a CUE package.
func Generate(inst *cue.Instance) (b []byte, err error) {
	if err = inst.Value().Validate(cue.ResolveReferences(false)); err != nil {
		return nil, err
	}

	g := &generator{
		typeMap: map[string]types.Type{},
	}
	// TODO select codegen logic to execute based on package-level attr (compare to: proto2, proto3)
	// TODO how the hell do we require the input artifacts to be versioned

	iter, err := inst.Value().Fields(cue.Definitions(true), cue.DisallowCycles(true), cue.ResolveReferences(false))
	if err != nil {
		os.Exit(1)
	}

	// TODO need a whole indirection layer here for a builder that can analyze,
	// validate, and prep outputs independent of actually generating templated
	// output.
	for iter.Next() {
		g.decl(iter.Label(), iter.Value())
	}

	if g.err != nil {
		return nil, err
	}

	return g.w.Bytes(), nil
}

type generator struct {
	typeMap map[string]types.Type

	w   bytes.Buffer
	err errors.Error
}

func (g *generator) addErr(err error) {
	if err != nil {
		g.err = errors.Append(g.err, errors.Promote(err, "generate failed"))
	}
}

func (g *generator) exec(t *template.Template, data interface{}) {
	g.addErr(t.Execute(&g.w, data))
}

func (g *generator) decl(name string, v cue.Value) {
	attr := v.Attribute("grafanats")
	// dumpJSON(name, v, false)

	// Establish that we have an exported (upper case first letter) name, and
	// that the expected grafanats attribute is present.
	if !ast.IsExported(name) || attr.Err() != nil {
		return
	}

	// Establish that the expected "targetType" property is present within the
	// grafanats attribute.
	tt, found, err := attr.Lookup(0, "targetType")
	if !found || err != nil {
		return
	}

	// Value preparation:
	// 1. Inspect for defaults, do...what with them?
	// 2. For strings, wrap in single quotes
	// 3. For basic types, retain as string literal
	// 4. For named types, deref to type if a definition (#-led), else translate to string literal
	// 5. Reject all field/list comprehensions?
	// 6. String interpolation probably shouldn't be allowed
	// 7. Probably can't allow any function calls either

	// Validation TODOs
	// - Disallow any nested structs beneath a struct literal (struct literal,
	//   because field comprehensions and files are represented in adt as
	//   structs). See Value.Source()
	// - Experiment with things like field comprehensions, string evals, etc.,
	//   to see how much evaluation we can easily trigger (and therefore, how
	//   little of CUE we have to cut off) without making unclear exactly what
	//   gets exported to TS
	// - See if we can write a CUE file for generalized validation of the inputs
	//   to this program - e.g., all enum values are lowerCamelCase

	type KV struct{ K, V string }
	var pairs []KV
	tvars := map[string]interface{}{
		"name":   name,
		"export": true,
	}

	switch tt {
	case "enum":
		// We restrict the expression of enums to disjunctions (sum types).
		op, dvals := v.Expr()
		if op != cue.OrOp {
			g.addErr(fmt.Errorf("Typescript enums may only be generated from a disjunction of strings"))
			return
		}

		for _, dv := range dvals {
			text, _ := dv.String()
			if !dv.IsConcrete() || dv.IncompleteKind() != cue.StringKind || err != nil {
				g.addErr(fmt.Errorf("Typescript enums may only be generated from a disjunction of strings"))
				return
			}
			// Simple mapping of all enum values - which we are assuming are in
			// lowerCamelCase - to corresponding CamelCase
			pairs = append(pairs, KV{K: strings.Title(text), V: fmt.Sprintf("'%s'", text)})
		}

		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].K < pairs[j].K
		})
		tvars["pairs"] = pairs

		// TODO comments, maturity
		g.exec(enumCode, tvars)
	case "interface":
		// We restrict the derivation of Typescript interfaces to struct kinds.
		// (More than just a plain "{...}" struct match this, though.)
		if v.IncompleteKind() != cue.StructKind {
			// TODO figure out how to attach cue token positions to errors
			g.addErr(fmt.Errorf("Typescript interfaces may only be generated from structs"))
			return
		}

		// Handle expressions directly on the label node itself
		op, dvals := v.Expr()
		switch op {
		case cue.AndOp:
			for _, dv := range dvals {
				s := dv.Source()
				_ = s
			}
		}

		iter, err := v.Fields(cue.Definitions(true))
		_, _ = iter, err
		for i, _ := v.Fields(cue.Definitions(true)); i.Next(); {
			if i.IsHidden() || i.IsDefinition() {
				continue
			}

			optional := ""
			if i.IsOptional() {
				optional = "?"
			}
			k := fmt.Sprintf("%s%s", i.Label(), optional)

			s := v.Source()
			_ = s

			// Handle expressions
			op, dvals := i.Value().Expr()
			switch op {
			case cue.AndOp:
				for _, dv := range dvals {
					s := dv.Source()
					_ = s
				}
			}

			// First, check kind.
			ik := i.Value().IncompleteKind()
			switch ik {
			case cue.StructKind:
				// Nested structs are not allowed (for now), but there are
				// several valid cases that have incomplete StructKind which are
				// acceptable.

				// TODO finish
			case cue.ListKind:
				// Only translateable to typescript if it's a 0-element open
				// list with a basic type, e.g. [...number] => number[]

				// TODO finish
			case cue.StringKind, cue.NumberKind, cue.FloatKind, cue.IntKind, cue.BoolKind:
				// For all of these scalar types, we only want to permit types - that is, non-concrete options.
				if !v.IsConcrete() {
					// TODO figure out how to attach cue token positions to errors
					g.addErr(fmt.Errorf("Only CUE basic types, not constraints or concrete values, may be used for interface generation"))
					return
				}
				pairs = append(pairs, KV{K: k, V: fmtTypeAsTypescript(ik)})
			case cue.BytesKind:
				// TODO figure out how to attach cue token positions to errors
				g.addErr(fmt.Errorf("Bytes have no equivalent in Typescript; use double-quotes (string) instead"))
				return
			default:
				// TODO differentiate between other types (is that necessary?) and sum types - popcount the bits?
				// TODO figure out how to attach cue token positions to errors
				g.addErr(fmt.Errorf("Unrecognized kind %s", ik))
				return
			}
		}

		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].K < pairs[j].K
		})
		tvars["pairs"] = pairs

		g.exec(interfaceCode, tvars)
	default:
		return // TODO error out
	}
}

func fmtTypeAsTypescript(k cue.Kind) string {
	switch k {
	case cue.BoolKind:
		return "boolean"
	case cue.StringKind, cue.NumberKind, cue.FloatKind, cue.IntKind:
		return k.String()
	default:
		return ""
	}
}

func fmtAtomAsTypescript(v cue.Value) string {
	switch v.Kind() {
	case cue.StringKind:
		// Shouldn't be possible for this to err on a concrete value
		str, _ := v.String()
		return fmt.Sprintf("'%s'", str)
	case cue.BoolKind:

	}

	return ""
}

func isStruct(v cue.Value) bool {
	_, err := v.Struct()
	return err == nil
}
