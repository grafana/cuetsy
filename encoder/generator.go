package encoder

import (
	"bytes"
	"fmt"
	gast "go/ast"
	"math/bits"
	"os"
	"sort"
	"strings"
	"text/template"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/errors"
)

const attrname = "cuetsy"
const attrkey = "targetType"

type attrTSTarget string

const (
	tgtType      attrTSTarget = "type"
	tgtInterface attrTSTarget = "interface"
	tgtEnum      attrTSTarget = "enum"
)

// TODO use txtar to set up a buncha test cases

// Generate takes a cue.Instance and generates the corresponding Typescript.
//
// It is expected that the cue.Instance represents a top-level struct - that is,
// the contents of a single file or merged contents of a CUE package.
func Generate(inst *cue.Instance) (b []byte, err error) {
	if err = inst.Value().Validate(); err != nil {
		return nil, err
	}

	g := &generator{
		inst: inst,
	}
	// TODO select codegen logic to execute based on package-level attr (compare to: proto2, proto3)
	// TODO how the hell do we require the input artifacts to be versioned

	iter, err := inst.Value().Fields(cue.Definitions(true))
	if err != nil {
		errors.Print(os.Stderr, err, &errors.Config{Cwd: "."})
		os.Exit(1)
	}

	for iter.Next() {
		g.decl(iter.Label(), iter.Value())
	}

	return g.w.Bytes(), g.err
}

type generator struct {
	inst *cue.Instance
	w    bytes.Buffer
	err  errors.Error
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
	dumpJSON(name, v, false)
	if !gast.IsExported(name) {
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
	// - Disallow struct literals nested within struct literals (?). (struct
	//   literal, because field comprehensions and files are represented in adt as
	//   structs. See Value.Source())
	// - Experiment with things like field comprehensions, string evals, etc.,
	//   to see how much evaluation we can easily trigger (and therefore, how
	//   little of CUE we have to cut off) without making unclear exactly what
	//   gets exported to TS
	// - See if we can write a CUE file for generalized validation of the inputs
	//   to this program - e.g., all enum values are lowerCamelCase
	// - Disallow exported structs without an annotation...? The only goal there would
	//   be to try to provide more guiding guardrails to users

	tst, err := getTSTarget(v)
	if err != nil {
		// Ignore values without attributes
		return
	}
	switch tst {
	case tgtEnum:
		g.genEnum(name, v)
		return
	case tgtInterface:
		g.genInterface(name, v)
		return
	case tgtType:
		g.genType(name, v)
		return
	default:
		return // TODO error out
	}
}

func (g *generator) genType(name string, v cue.Value) {
	tvars := map[string]interface{}{
		"name":   name,
		"export": true,
	}

	var tokens []string
	op, dvals := v.Expr()
	switch op {
	case cue.OrOp:
		for _, dv := range dvals {
			tok, err := tsprintField(dv)
			if err != nil {
				g.addErr(err)
				return
			}
			tokens = append(tokens, tok)
		}
	case cue.NoOp:
		tok, err := tsprintField(v)
		if err != nil {
			g.addErr(err)
			return
		}
		tokens = append(tokens, tok)
	default:
		g.addErr(valError(v, "typescript types may only be generated from a single value or disjunction of values"))
	}

	tvars["tokens"] = tokens

	// TODO comments
	// TODO maturity marker (@alpha, etc.)
	g.exec(typeCode, tvars)
}

func (g *generator) genEnum(name string, v cue.Value) {
	type KV struct{ K, V string }
	var pairs []KV
	tvars := map[string]interface{}{
		"name":   name,
		"export": true,
	}

	// We restrict the expression of TS enums to CUE disjunctions (sum types) of strings.
	op, dvals := v.Expr()
	if op != cue.OrOp || v.IncompleteKind() != cue.StringKind {
		// Note: IncompleteKind() returns the union of basic types present
		// within the set of values in the sum type. Thus, the sum type's
		// IncompleteKind() being string tells us that all elements of the
		// sum type are a string (they have string as their least upper
		// bound).
		g.addErr(fmt.Errorf("typescript enums may only be generated from a disjunction of strings"))
		return
	}

	for _, dv := range dvals {
		text, _ := dv.String()
		if !dv.IsConcrete() {
			g.addErr(fmt.Errorf("typescript enums may only be generated from a disjunction of concrete strings"))
			return
		}
		// Simple mapping of all enum values (which we are assuming are in
		// lowerCamelCase) to corresponding CamelCase
		pairs = append(pairs, KV{K: strings.Title(text), V: tsprintConcrete(dv)})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].K < pairs[j].K
	})
	tvars["pairs"] = pairs

	// TODO comments
	// TODO maturity marker (@alpha, etc.)
	g.exec(enumCode, tvars)
}

func (g *generator) genInterface(name string, v cue.Value) {
	type KV struct{ K, V string }
	var pairs []KV
	tvars := map[string]interface{}{
		"name":    name,
		"export":  true,
		"extends": []string{},
	}

	// We restrict the derivation of Typescript interfaces to struct kinds.
	// (More than just a struct literal match this, though.)
	if v.IncompleteKind() != cue.StructKind {
		// TODO figure out how to attach cue token positions to errors
		g.addErr(fmt.Errorf("typescript interfaces may only be generated from structs"))
		return
	}

	// There are basic two paths to extracting what we treat as the body
	// of the Typescript interface to generate. The first, simpler case,
	// applies when there's just a literal struct declaration for the label,
	// e.g.:
	//
	//  a: {
	//	  foo: string
	//  }
	//
	// Such declarations return an empty []Value from Expr(), so we
	// construct them through Value.Fields() instead. However, when there's
	// unification involved:
	//
	// b: a & {
	//	  bar: string
	//  }
	//
	// Then Value.Fields() represents the *results* of evaluating the
	// expression. This is an unavoidable part of constructing the value
	// (cue.Instance.Value() triggers it), but it's not what we want for
	// generating Typescript. Our goal is to generate text such that
	// Typescript will reach the same final semantics as CUE does, but
	// through its own evaluation, rather than just spitting out the final
	// result. (In other words, we want the TS and CUE to look structurally
	// the same-ish.) So, if Value.Expr() returns at least one result, we
	// call it continuously until we find a Value from an ast.StructLit,
	// which contains only the literal declarations in its Fields().
	//
	// TODO The exception is if we find definitions in the Expr Values,
	// which must then be directly unified into the struct literal.
	fields, err := v.Fields(cue.Optional(true))
	if err != nil {
		panic("unreachable: already verified we have a StructKind?")
	}

	op, _ := v.Expr()
	if op != cue.NoOp {
		var extends []string
		var foundLiteral bool

		// Recursively walk down Expr() return Values and pull out the interesting ones
		var walkExpr func(v cue.Value, belowAnd bool) error
		walkExpr = func(v cue.Value, belowAnd bool) error {
			op, dvals := v.Expr()
			switch op {
			case cue.NoOp:
				return nil
			case cue.OrOp:
				return valError(v, "typescript interfaces cannot be constructed from disjunctions")
			case cue.SelectorOp:
				if !belowAnd {
					// Only (?) interested in this to extract name of unified struct, if we're under a conjunction
					return nil
				}
				if len(dvals) != 2 {
					return valError(v, "selector expressions should have two operands; wtf")
				}

				// This gives us the string value of the identifier being
				// merged, so we can look it up and retrieve its attributes.
				// TODO what if it's a nested struct? Will this still work for a path lookup? uuughhh
				label, err := dvals[1].String()
				if err != nil {
					return err
				}
				lv := g.inst.Lookup(label)
				if !lv.Exists() {
					return valError(dvals[1], "should be unreachable, as the identifier must have a valid referent to pass earlier validation")
				}
				// TODO An error is probably right, but is there an argument to
				// be made that this should fall back to just merging in, as a
				// definition would?
				if !checkTSTarget(tgtInterface, lv) {
					return valError(dvals[1], "interface-targeted structs may only be unified with other structs that target interfaces")
				}
				extends = append(extends, label)
				return nil
			case cue.AndOp:
				// First, search the dvals for a StructLit. That'll be the only one we have to deal with.
				for _, dv := range dvals {
					if dv.IncompleteKind() != cue.StructKind {
						panic("impossible? seems like it should be. if this pops, clearly not!")
					}
					// We go depth-first, as the LHS of a series of unifications
					// over structs gets incrementally populated with each
					// unified struct as you move up from the leaves. (Wait,
					// does this actually matter?)
					if err := walkExpr(dv, true); err != nil {
						return err
					}
					if _, ok := dv.Source().(*ast.StructLit); ok {
						var err error
						fields, err = dv.Fields(cue.Optional(true))
						// TODO error if we find more than one?
						foundLiteral = true
						if err != nil {
							return err
						}
					}
				}
				return nil
			}
			return nil
		}
		if err := walkExpr(v, false); err != nil {
			g.addErr(err)
			return
		}
		if !foundLiteral {
			fields = nil
		}
		tvars["extends"] = extends
	}

	// We now have an iterator that represents the set of fields we want to
	// place in the body of the generated typescript interface. (Or nil, if
	// there's no body to generate.)
	for fields != nil && fields.Next() {
		if fields.IsHidden() {
			// TODO figure out how to attach cue token positions to errors
			g.addErr(valError(fields.Value(), "cannot generate hidden fields; typescript has no corresponding concept"))
			return
		}

		k := fields.Label()
		if fields.IsOptional() {
			k += "?"
		}

		vstr, err := tsprintField(fields.Value())
		if err != nil {
			g.addErr(err)
			return
		}
		pairs = append(pairs, KV{K: k, V: vstr})
	}

	sort.Slice(pairs, func(i, j int) bool { return pairs[i].K < pairs[j].K })
	tvars["pairs"] = pairs

	g.exec(interfaceCode, tvars)
}

// Render a string containing a Typescript semantic equivalent to the provided
// Value, if possible.
//
// The provided Value must be a simple expression (loosely defined, until
// something more precise is understood); e.g., this will NOT render a struct
// literal.
func tsprintField(v cue.Value) (string, error) {
	// References appear to be largely orthogonal to the Kind system. Handle them first.
	_, path := v.Reference()
	if len(path) > 0 {
		return strings.Join(path, "."), nil
	}

	op, dvals := v.Expr()
	// Eliminate concretes first, to make handling the others easier.
	switch v.Kind() {
	case cue.StructKind:
		switch s := v.Source().(type) {
		case *ast.StructLit:
			return "", valError(v, "nested structs are not yet supported")
		case *ast.Field:
			// TODO
			// return s.Label
			// Otherwise it's gonna (?) be a field, and we just print its name,
			// which should be available via String() of the second op from Expr()
			_ = s
			if op != cue.SelectorOp {
				panic("fixme")
			}
			return dvals[1].String()
		default:
			panic("wtf")
		}
	case cue.ListKind:
		// A list is concrete (and thus its complete kind is ListKind instead of
		// BottomKind) iff it specifies a finite number of elements - is
		// "closed". This is independent of the types of its elements, which may
		// be anywhere on the concreteness spectrum.
		//
		// For closed lists, we simply iterate over its component elements and
		// print their typescript representation.
		iter, _ := v.List()
		var parts []string
		for iter.Next() {
			part, err := tsprintField(iter.Value())
			if err != nil {
				return "", err
			}
			parts = append(parts, part)
		}
		return fmt.Sprintf("[%s]", strings.Join(parts, ", ")), nil
	case cue.StringKind, cue.BoolKind, cue.FloatKind, cue.IntKind:
		return tsprintConcrete(v), nil
	case cue.BytesKind:
		return "", valError(v, "bytes have no equivalent in Typescript; use double-quotes (string) instead")
	}

	// Handler for disjunctions
	disj := func(dvals []cue.Value) (string, error) {
		parts := make([]string, 0, len(dvals))
		for _, dv := range dvals {
			p, err := tsprintField(dv)
			if err != nil {
				return "", err
			}
			parts = append(parts, p)
		}
		return strings.Join(parts, " | "), nil
	}

	ik := v.IncompleteKind()
	switch ik {
	case cue.BottomKind:
		return "", valError(v, "bottom, unsatisfiable")
	case cue.ListKind:
		// This list is open - its final element is ...<value> - and we can only
		// meaningfully convert open lists to typescript if there are zero other
		// elements.
		e, has := v.Elem()
		if !has {
			panic("unreachable - non-concrete list should entail Elem() returns something")
		}
		elemstr, err := tsprintField(e)
		if err != nil {
			return "", err
		}

		// Verify there are no other list elements.
		iter, _ := v.List()
		// TODO There's gotta be a better way of checking this
		for iter.Next() {
			return "", valError(v, "open lists are only supported with zero values; try as [...%s]", elemstr)
		}
		return elemstr + "[]", nil // TODO
	case cue.NumberKind, cue.StringKind:
		// It appears there are only three cases in which we can have an
		// incomplete NumberKind or StringKind:
		//
		// 1. The corresponding literal is a bounding constraint (which subsumes
		// both int and float), e.g. >2.2, <"foo"
		// 2. There's a disjunction of concrete literals of the relevant type
		// 2. The corresponding literal is the basic type "number" or "string"
		//
		// The former case has no equivalent in typescript, but the latter is
		// easy. We disambiguate by seeing if there is an expression, which is
		// how ">" and "2.2" are bound together.
		//
		// TODO get more certainty/a clearer way of ascertaining this
		if op != cue.NoOp {
			return "", valError(v, "bounds constraints are not supported as they lack a direct typescript equivalent")
		}
		fallthrough
	case cue.FloatKind, cue.IntKind, cue.BoolKind, cue.NullKind:
		// Having eliminated the possibility of bounds/constraints, we're left
		// with disjunctions and basic types.
		if op == cue.OrOp {
			// It's a disjunction.
			return disj(dvals)
		} else if op != cue.NoOp {
			panic("unreachable...?")
		}
		// There's no op; it's a basic type, and can be trivially rendered.
		fallthrough
	case cue.TopKind:
		return tsprintType(ik), nil
	}

	// Having more than one possible kind entails a disjunction, TopKind, or
	// NumberKind. We've already eliminated TopKind and NumberKind, so now check
	// if there's more than one bit set. (If there isn't, it's a bug: we've
	// missed a kind above). If so, run our disjunction-handling logic.
	if bits.OnesCount16(uint16(ik)) > 1 {
		return disj(dvals)
	}

	return "", valError(v, "unrecognized kind %s", ik)
}

// ONLY call this function if it has been established that the provided Value is
// Concrete.
func tsprintConcrete(v cue.Value) string {
	switch v.Kind() {
	case cue.NullKind:
		return "null"
	case cue.StringKind:
		s, _ := v.String()
		return fmt.Sprintf("'%s'", s)
	case cue.FloatKind:
		f, _ := v.Float64()
		return fmt.Sprintf("%g", f)
	case cue.NumberKind, cue.IntKind:
		i, _ := v.Int64()
		return fmt.Sprintf("%v", i)
	case cue.BoolKind:
		if b, _ := v.Bool(); b {
			return "true"
		}
		return "false"
	default:
		panic("unreachable")
	}
}

func tsprintType(k cue.Kind) string {
	switch k {
	case cue.BoolKind:
		return "boolean"
	case cue.StringKind:
		return "string"
	case cue.NumberKind, cue.FloatKind, cue.IntKind:
		return "number"
	case cue.TopKind:
		return "any"
	default:
		return ""
	}
}

func getTSTarget(v cue.Value) (attrTSTarget, error) {
	a := v.Attribute(attrname)
	if a.Err() != nil {
		return "", a.Err()
	}

	tt, found, err := a.Lookup(0, "targetType")
	if !found {
		return "", valError(v, "no value for the %q key in @%s attribute", attrkey, attrname)
	}
	if err != nil {
		return "", err
	}

	return attrTSTarget(tt), nil
}

// Checks if the supplied Value has an attribute indicating the given targetAttr
func checkTSTarget(t attrTSTarget, v cue.Value) bool {
	tt, err := getTSTarget(v)
	if err != nil {
		return false
	}

	return tt == t
}

func valError(v cue.Value, format string, args ...interface{}) error {
	s := v.Source()
	if s == nil {
		return fmt.Errorf(format, args...)
	}
	return errors.Newf(s.Pos(), format, args...)
}

func isReference(v cue.Value) bool {
	_, path := v.Reference()
	return len(path) > 0
}
