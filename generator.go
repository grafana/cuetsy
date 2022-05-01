package cuetsy

import (
	"bytes"
	"fmt"
	"go/token"
	"math/bits"
	"sort"
	"strings"
	"text/template"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/errors"
	"github.com/grafana/cuetsy/ts"
	tsast "github.com/grafana/cuetsy/ts/ast"
)

const (
	attrname        = "cuetsy"
	attrEnumDefault = "enumDefault"
	attrEnumMembers = "memberNames"
	attrKind        = "kind"
	attrForceText   = "forceText"
)

type tsKind string

const (
	kindType      tsKind = "type"
	kindInterface tsKind = "interface"
	kindEnum      tsKind = "enum"
)

var allKinds = [...]tsKind{
	kindType,
	kindInterface,
	kindEnum,
}

// An ImportMapper takes an ImportDecl and returns a string indicating the
// import statement that should be used in the corresponding typescript, or
// an error if no mapping can be made.
type ImportMapper func(*ast.ImportDecl) (string, error)

// NoImportMappingErr returns a standard error indicating that no mapping can be
// made for the provided import statement.
func NoImportMappingErr(d *ast.ImportDecl) error {
	return errors.Newf(d.Pos(), "a corresponding typescript import is not available for %q", d.Import.String())
}

func nilImportMapper(d *ast.ImportDecl) (string, error) { return "", NoImportMappingErr(d) }

// Config governs certain variable behaviors when converting CUE to Typescript.
type Config struct {
	// ImportMapper determines how CUE imports are mapped to Typescript imports.
	// If nil, any non-stdlib import in the CUE source will result in a fatal
	// error.
	ImportMapper
}

// Generate takes a cue.Value and generates the corresponding TypeScript for all
// top-level members of that value that have appropriate @cuetsy attributes.
//
// Members that are definitions, hidden fields, or optional fields are ignored.
func Generate(val cue.Value, c Config) (b []byte, err error) {
	file, err := GenerateAST(val, c)
	if err != nil {
		return nil, err
	}
	return []byte("\n" + file.String()), nil
}

func GenerateAST(val cue.Value, c Config) (*ts.File, error) {
	if err := val.Validate(); err != nil {
		return nil, err
	}

	if c.ImportMapper == nil {
		c.ImportMapper = nilImportMapper
	}

	g := &generator{
		c:   c,
		val: &val,
	}

	iter, err := val.Fields(
		cue.Definitions(true),
		cue.Concrete(false),
	)
	if err != nil {
		return nil, err
	}

	var file ts.File
	for iter.Next() {
		n := g.decl(iter.Label(), iter.Value())
		file.Nodes = append(file.Nodes, n...)
	}

	return &file, g.err
}

type generator struct {
	val *cue.Value
	c   Config
	err errors.Error
}

func (g *generator) addErr(err error) {
	if err != nil {
		g.err = errors.Append(g.err, errors.Promote(err, "generate failed"))
	}
}

func execGetString(t *template.Template, data interface{}) (string, error) {
	var tpl bytes.Buffer
	if err := t.Execute(&tpl, data); err != nil {
		return "", err
	}
	result := tpl.String()
	return result, nil
}

func (g *generator) decl(name string, v cue.Value) []ts.Decl {
	if !token.IsExported(name) {
		return nil
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
	// - Experiment with things like field comprehensions, string evals, etc.,
	//   to see how much evaluation we can easily trigger (and therefore, how
	//   little of CUE we have to cut off) without making unclear exactly what
	//   gets exported to TS
	// - See if we can write a CUE file for generalized validation of the inputs
	//   to this program - e.g., all enum values are lowerCamelCase
	// - Disallow exported structs without an annotation...? The only goal there would
	//   be to try to provide more guiding guardrails to users

	tst, err := getKindFor(v)
	if err != nil {
		// Ignore values without attributes
		return nil
	}
	switch tst {
	case kindEnum:
		return g.genEnum(name, v)
	case kindInterface:
		return g.genInterface(name, v)
	case kindType:
		return g.genType(name, v)
	default:
		return nil // TODO error out
	}
}

func (g *generator) genType(name string, v cue.Value) []ts.Decl {
	var tokens []tsast.Expr
	// If there's an AndOp first, pass through it.
	op, dvals := v.Expr()
	if op == cue.AndOp {
		op, dvals = dvals[0].Expr()
	}
	switch op {
	case cue.OrOp:
		for _, dv := range dvals {
			tok, err := tsprintField(dv)
			if err != nil {
				g.addErr(err)
				return nil
			}
			tokens = append(tokens, tok)
		}
	case cue.NoOp:
		tok, err := tsprintField(v)
		if err != nil {
			g.addErr(err)
			return nil
		}
		tokens = append(tokens, tok)
	default:
		g.addErr(valError(v, "typescript types may only be generated from a single value or disjunction of values"))
	}

	T := ts.Export(
		tsast.TypeDecl{
			Name: ts.Ident(name),
			Type: tsast.BasicType{Expr: ts.Union(tokens...)},
		},
	)

	d, ok := v.Default()
	if !ok {
		return []ts.Decl{T}
	}

	val, err := tsprintField(d)
	g.addErr(err)

	D := ts.Export(
		tsast.VarDecl{
			Names: ts.Names("default" + name),
			Type:  ts.Ident(name),
			Value: val,
		},
	)

	return []ts.Decl{T, D}
}

type KV struct {
	K, V string
}

// genEnum turns the following cue values into typescript enums:
// - value disjunction (a | b | c): values are taken as attribut memberNames,
//   if memberNames is absent, then keys implicitely generated as CamelCase
// - string struct: struct keys get enum keys, struct values enum values
func (g *generator) genEnum(name string, v cue.Value) []ts.Decl {
	// FIXME compensate for attribute-applying call to Unify() on incoming Value
	op, dvals := v.Expr()
	if op == cue.AndOp {
		v = dvals[0]
		op, _ = v.Expr()
	}

	// We restrict the expression of TS enums to CUE disjunctions (sum types) of strings.
	allowed := cue.StringKind | cue.NumberKind | cue.NumberKind
	ik := v.IncompleteKind()
	if op != cue.OrOp || ik&allowed != ik {
		g.addErr(valError(v, "typescript enums may only be generated from a disjunction of concrete int with memberNames attribute or strings"))
		return nil
	}

	exprs, err := orEnum(v)
	if err != nil {
		g.addErr(err)
	}

	T := ts.Export(
		tsast.TypeDecl{
			Name: ts.Ident(name),
			Type: tsast.EnumType{Elems: exprs},
		},
	)

	defaultIdent, err := enumDefault(v)
	if err != nil {
		g.addErr(err)
	}

	if defaultIdent == nil {
		return []ts.Decl{T}
	}

	D := ts.Export(
		tsast.VarDecl{
			Names: ts.Names("default" + name),
			Type:  ts.Ident(name),
			Value: tsast.SelectorExpr{Expr: ts.Ident(name), Sel: *defaultIdent},
		},
	)
	return []ts.Decl{T, D}
}

func enumDefault(v cue.Value) (*tsast.Ident, error) {
	def, ok := v.Default()
	if !ok {
		return nil, def.Err()
	}

	if v.IncompleteKind() == cue.StringKind {
		s, _ := def.String()
		return &tsast.Ident{Name: strings.Title(s)}, nil
	}

	// For Int, Float, Numeric we need to find the default value and its corresponding memberName value
	a := v.Attribute(attrname)
	val, found, err := a.Lookup(0, attrEnumMembers)
	if err != nil || !found {
		panic(fmt.Sprintf("looking up memberNames: found=%t err=%s", found, err))
	}
	evals := strings.Split(val, "|")

	_, dvals := v.Expr()
	for i, val := range dvals {
		valLab, _ := val.Label()
		defLab, _ := def.Label()
		if valLab == defLab {
			return &tsast.Ident{Name: evals[i]}, nil
		}
	}

	// should never reach here tho
	return nil, valError(v, "unable to find memberName corresponding to the default")
}

func orEnum(v cue.Value) ([]ts.Expr, error) {
	_, dvals := v.Expr()
	a := v.Attribute(attrname)

	var attrMemberNameExist bool
	var evals []string
	if a.Err() == nil {
		val, found, err := a.Lookup(0, attrEnumMembers)
		if err == nil && found {
			attrMemberNameExist = true
			evals = strings.Split(val, "|")
			if len(evals) != len(dvals) {
				return nil, valError(v, "typescript enums and %s attributes size doesn't match", attrEnumMembers)
			}
		}
	}

	// We only allowed String Enum to be generated without memberName attribute
	if v.IncompleteKind() != cue.StringKind && !attrMemberNameExist {
		return nil, valError(v, "typescript numeric enums may only be generated from memberNames attribute")
	}

	var fields []ts.Expr
	for idx, dv := range dvals {
		var text string
		if attrMemberNameExist {
			text = evals[idx]
		} else {
			text, _ = dv.String()
		}

		if !dv.IsConcrete() {
			return nil, valError(v, "typescript enums may only be generated from a disjunction of concrete strings")
		}

		fields = append(fields, tsast.AssignExpr{
			// Simple mapping of all enum values (which we are assuming are in
			// lowerCamelCase) to corresponding CamelCase
			Name:  ts.Ident(strings.Title(text)),
			Value: tsprintConcrete(dv),
		})
	}

	sort.Slice(fields, func(i, j int) bool {
		return fields[i].String() < fields[j].String()
	})

	return fields, nil
}

func (g *generator) genInterface(name string, v cue.Value) []ts.Decl {
	// We restrict the derivation of Typescript interfaces to struct kinds.
	// (More than just a struct literal match this, though.)
	if v.IncompleteKind() != cue.StructKind {
		// FIXME check for bottom here, give different error
		g.addErr(valError(v, "typescript interfaces may only be generated from structs"))
		return nil
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
	// generating Typescript; cuetsy's goal is to generate Typescript text that
	// is semantically equivalent to the original CUE, but relying on
	// Typescript's "extends" composition where possible. This is necessary to
	// allow CUE subcomponents that are imported from dependencies which may
	// change to be "updated" in Typescript side through standard dependency
	// management means, rather than requiring a regeneration of the literal
	// type from CUE. (In other words, we want the TS text and CUE text to look
	// structurally the same-ish.)
	//
	// So, if Value.Expr() returns at least one result, we process the expression
	// parts to separate them into elements that should be literals on the
	// resulting Typescript interface, vs. ones that are composed via "extends."

	// Create an empty value, onto which we'll unify fields that need not be
	// generated as literals.
	nolit := v.Context().CompileString("{...}")

	var extends []ts.Expr
	var some bool

	// Recursively walk down Values returned from Expr() and separate
	// unified/embedded structs from a struct literal, so that we can make the
	// former (if they are also marked with @cuetsy(kind="interface")) show up
	// as "extends" instead of writing out their fields directly.
	var walkExpr func(wv cue.Value) error
	walkExpr = func(wv cue.Value) error {
		op, dvals := wv.Expr()
		switch op {
		case cue.NoOp:
			// Simple path - when the field is a plain struct literal decl, the walk function
			// will take this branch and return immediately.

			// FIXME this does the struct literal path correctly, but it also
			// catches this case, for some reason:
			//
			//   Thing: {
			//       other.Thing
			//   }
			//
			// The saner form - `Thing: other.Thing` - does not go through this path.
			return nil
		case cue.OrOp:
			return valError(wv, "typescript interfaces cannot be constructed from disjunctions")
		case cue.SelectorOp:
			expr, err := referenceValueAs(wv, kindInterface)
			if err != nil {
				return err
			}

			// If we have a string to add to the list of "extends", then also
			// add the ref to the list of fields to exclude if subsumed.
			if expr != nil {
				some = true
				extends = append(extends, expr)
				nolit = nolit.Unify(cue.Dereference(wv))
			}
			return nil
		case cue.AndOp:
			// First, search the dvals for StructLits. Having more than one is possible,
			// but weird, as writing >1 literal and unifying them is the same as just writing
			// one containing the unified result - more complicated with no obvious benefit.
			for _, dv := range dvals {
				if dv.IncompleteKind() != cue.StructKind {
					panic("impossible? seems like it should be. if this pops, clearly not!")
				}

				if err := walkExpr(dv); err != nil {
					return err
				}
			}
			return nil
		default:
			panic(fmt.Sprintf("unhandled op type %s", op.String()))
		}
	}

	if err := walkExpr(v); err != nil {
		g.addErr(err)
		return nil
	}
	var elems []tsast.KeyValueExpr
	var defs []tsast.KeyValueExpr

	iter, _ := v.Fields(cue.Optional(true))
	for iter != nil && iter.Next() {
		if iter.Selector().PkgPath() != "" {
			g.addErr(valError(iter.Value(), "cannot generate hidden fields; typescript has no corresponding concept"))
			return nil
		}

		// Skip fields that are subsumed by the Value representing the
		// unification of all refs that will be represented using an "extends"
		// keyword.
		//
		// This does introduce the possibility that even some fields which are
		// literally declared on the struct will not end up written out in
		// Typescript (though the semantics will still be correct). That's
		// likely to be a bit confusing for users, but we have no choice. The
		// (preferable) alternative would rely on Unify() calls to build a Value
		// containing only those fields that we want, then iterating over that
		// in this loop.
		//
		// Unfortunately, as of v0.4.0, Unify() appears to not preserve
		// attributes on the Values it generates, which makes it impossible to
		// rely on, as the tsprintField() func later also needs to check these
		// attributes in order to decide whether to render a field as a
		// reference or a literal.
		//
		// There's _probably_ a way around this, especially when we move to an
		// AST rather than dumb string templates. But i'm tired of looking.
		if some {
			// Look up the path of the current field within the nolit value,
			// then check it for subsumption.
			sel := iter.Selector()
			if iter.IsOptional() {
				sel = sel.Optional()
			}
			sub := nolit.LookupPath(cue.MakePath(sel))

			// Theoretically, lattice equality can be defined as bijective
			// subsumption. In practice, Subsume() seems to ignore optional
			// fields, and Equals() doesn't. So, use Equals().
			if sub.Exists() && sub.Equals(iter.Value()) {
				continue
			}
		}

		k := iter.Selector().String()
		if iter.IsOptional() {
			k += "?"
		}

		expr, err := tsprintField(iter.Value())
		if err != nil {
			g.addErr(err)
			return nil
		}

		elems = append(elems, tsast.KeyValueExpr{
			Key:   ts.Ident(k),
			Value: expr,
		})

		exists, defExpr, err := tsPrintDefault(iter.Value())
		g.addErr(err)

		if !exists {
			continue
		}

		defs = append(defs, tsast.KeyValueExpr{
			Key:   ts.Ident(strings.TrimSuffix(k, "?")),
			Value: defExpr,
		})
	}

	sort.Slice(elems, func(i, j int) bool {
		return elems[i].Key.String() < elems[j].Key.String()
	})
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].Key.String() < defs[j].Key.String()
	})

	T := ts.Export(
		tsast.TypeDecl{
			Name: ts.Ident(name),
			Type: tsast.InterfaceType{
				Elems:   elems,
				Extends: extends,
			},
		},
	)

	if len(defs) == 0 {
		return []ts.Decl{T}
	}

	D := ts.Export(
		tsast.VarDecl{
			Names: ts.Names("default" + name),
			Type:  ts.Ident(name),
			Value: tsast.ObjectLit{Elems: defs},
		},
	)

	return []ts.Decl{T, D}
}

func tsPrintDefault(v cue.Value) (bool, ts.Expr, error) {
	d, ok := v.Default()
	// [...number] results in [], which is a fake default, we need to correct it here.
	if ok && d.Kind() == cue.ListKind {
		len, err := d.Len().Int64()
		if err != nil {
			return false, nil, err
		}
		var defaultExist bool
		if len <= 0 {
			op, vals := v.Expr()
			if op == cue.OrOp {
				for _, val := range vals {
					vallen, _ := d.Len().Int64()
					if val.Kind() == cue.ListKind && vallen <= 0 {
						defaultExist = true
						break
					}
				}
				if !defaultExist {
					ok = false
				}
			} else {
				ok = false
			}
		}
	}

	if ok {
		expr, err := tsprintField(d)
		if err != nil {
			return false, nil, err
		}

		if isReference(d) {
			switch t := expr.(type) {
			case tsast.SelectorExpr:
				t.Sel.Name = "default" + t.Sel.Name
				expr = t
			case tsast.Ident:
				t.Name = "default" + t.Name
				expr = t
			default:
				panic(fmt.Sprintf("unexpected type %T", expr))
			}
		}

		return true, expr, nil
	}
	return false, nil, nil
}

// Render a string containing a Typescript semantic equivalent to the provided
// Value for placement in a single field, if possible.
func tsprintField(v cue.Value) (ts.Expr, error) {
	// Let the forceText attribute supersede everything.
	if ft := getForceText(v); ft != "" {
		return ts.Raw(ft), nil
	}

	// References are orthogonal to the Kind system. Handle them first.
	ref, err := referenceValueAs(v)
	if err != nil {
		return nil, err
	}
	if ref != nil {
		return ref, nil
	}

	verr := v.Validate(cue.Final())
	if verr != nil {
		return nil, verr
	}

	op, dvals := v.Expr()
	// Eliminate concretes first, to make handling the others easier.

	// Concrete values.
	// Includes "foobar", 5, [1,2,3], etc. (literal values)
	k := v.Kind()
	switch k {
	case cue.StructKind:
		switch op {
		case cue.SelectorOp, cue.AndOp, cue.NoOp:
			iter, err := v.Fields()
			if err != nil {
				return nil, valError(v, "something went wrong when generate nested structs")
			}

			size, _ := v.Len().Int64()
			kvs := make([]tsast.KeyValueExpr, 0, size)
			for iter.Next() {
				expr, err := tsprintField(iter.Value())
				if err != nil {
					return nil, valError(v, err.Error())
				}

				kvs = append(kvs, tsast.KeyValueExpr{
					Key:   ts.Ident(iter.Label()),
					Value: expr,
				})
			}

			return tsast.ObjectLit{Elems: kvs}, nil
		default:
			panic(fmt.Sprintf("not expecting op type %d", op))
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
		var elems []ts.Expr
		for iter.Next() {
			e, err := tsprintField(iter.Value())
			if err != nil {
				return nil, err
			}
			elems = append(elems, e)
		}
		return ts.List(elems...), nil
	case cue.StringKind, cue.BoolKind, cue.FloatKind, cue.IntKind:
		return tsprintConcrete(v), nil
	case cue.BytesKind:
		return nil, valError(v, "bytes have no equivalent in Typescript; use double-quotes (string) instead")
	}

	// Handler for disjunctions
	disj := func(dvals []cue.Value) (ts.Expr, error) {
		parts := make([]ts.Expr, 0, len(dvals))
		for _, dv := range dvals {
			p, err := tsprintField(dv)
			if err != nil {
				return nil, err
			}
			parts = append(parts, p)
		}
		return ts.Union(parts...), nil
	}

	// Others: disjunctions, etc.
	ik := v.IncompleteKind()
	switch ik {
	case cue.BottomKind:
		return nil, valError(v, "bottom, unsatisfiable")
	case cue.ListKind:
		// This list is open - its final element is ...<value> - and we can only
		// meaningfully convert open lists to typescript if there are zero other
		// elements.
		e := v.LookupPath(cue.MakePath(cue.AnyIndex))
		has := e.Exists()
		if has {
			expr, err := tsprintField(e)
			if err != nil {
				return nil, err
			}
			return tsast.ListExpr{Expr: expr}, nil
		} else {
			// When it is a concrete list.
			iter, _ := v.List()
			if iter.Next() {
				expr := tsprintType(iter.Value().Kind())
				if expr == nil {
					label, _ := v.Label()
					return nil, valError(v, "can't convert list element of %v to typescript", label)
				}
				return tsast.ListExpr{Expr: expr}, nil
			}

			panic("💩")
		}
	case cue.NumberKind, cue.StringKind:
		// It appears there are only three cases in which we can have an
		// incomplete NumberKind or StringKind:
		//
		// 1. The corresponding literal is a bounding constraint (which subsumes
		// both int and float), e.g. >2.2, <"foo"
		// 2. There's a disjunction of concrete literals of the relevant type
		// 2. The corresponding literal is the basic type "number" or "string"
		//
		// The first case has no equivalent in typescript, and entails we error
		// out. The other two have the same handling as for other kinds, so we
		// fall through. We disambiguate by seeing if there is an expression
		// (other than Or, "|"), which is how ">" and "2.2" are represented.
		//
		// TODO get more certainty/a clearer way of ascertaining this
		switch op {
		case cue.NoOp, cue.OrOp, cue.AndOp:
		default:
			return nil, valError(v, "bounds constraints are not supported as they lack a direct typescript equivalent")
		}
		fallthrough
	case cue.FloatKind, cue.IntKind, cue.BoolKind, cue.NullKind:
		// Having eliminated the possibility of bounds/constraints, we're left
		// with disjunctions and basic types.
		switch op {
		case cue.OrOp:
			return disj(dvals)
		case cue.NoOp, cue.AndOp:
			// There's no op for simple unification; it's a basic type, and can
			// be trivially rendered.
		default:
			panic("unreachable...?")
		}
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

	return nil, valError(v, "unrecognized kind %s", ik)
}

// ONLY call this function if it has been established that the provided Value is
// Concrete.
func tsprintConcrete(v cue.Value) ts.Expr {
	switch v.Kind() {
	case cue.NullKind:
		return ts.Null()
	case cue.StringKind:
		s, _ := v.String()
		return ts.Str(s)
	case cue.FloatKind:
		f, _ := v.Float64()
		return ts.Float(f)
	case cue.NumberKind, cue.IntKind:
		i, _ := v.Int64()
		return ts.Int(i)
	case cue.BoolKind:
		b, _ := v.Bool()
		return ts.Bool(b)
	default:
		panic("unreachable")
	}
}

func tsprintType(k cue.Kind) ts.Expr {
	switch k {
	case cue.BoolKind:
		return ts.Ident("boolean")
	case cue.StringKind:
		return ts.Ident("string")
	case cue.NumberKind, cue.FloatKind, cue.IntKind:
		return ts.Ident("number")
	case cue.TopKind:
		return ts.Ident("any")
	default:
		return nil
	}
}

func valError(v cue.Value, format string, args ...interface{}) error {
	s := v.Source()
	if s == nil {
		return fmt.Errorf(format, args...)
	}
	return errors.Newf(s.Pos(), format, args...)
}

func refAsInterface(v cue.Value) (ts.Expr, error) {
	// Bail out right away if the value isn't a reference
	op, dvals := v.Expr()
	if !isReference(v) || op != cue.SelectorOp {
		return nil, fmt.Errorf("not a reference")
	}

	// Have to do attribute checks on the referenced field itself, so deref
	deref := cue.Dereference(v)
	dstr, _ := dvals[1].String()

	// FIXME It's horrifying, teasing out the type of selector kinds this way. *Horrifying*.
	switch dvals[0].Source().(type) {
	case nil:
		// A nil subject means an unqualified selector (no "."
		// literal).  This can only possibly be a reference to some
		// sibling or parent of the top-level Value being generated.
		// (We can't do cycle detection with the meager tools
		// exported in cuelang.org/go/cue, so all we have for the
		// parent case is hopium.)
		if _, ok := dvals[1].Source().(*ast.Ident); ok && targetsKind(deref, kindInterface) {
			return ts.Ident(dstr), nil
		}
	case *ast.SelectorExpr:
		// panic("case 2")
		if targetsKind(deref, kindInterface) {
			return ts.Ident(dstr), nil
		}
	case *ast.Ident:
		// panic("case 3")
		if targetsKind(deref, kindInterface) {
			str, ok := dvals[0].Source().(fmt.Stringer)
			if !ok {
				panic("expected dvals[0].Source() to implement String()")
			}

			return tsast.SelectorExpr{
				Expr: ts.Ident(str.String()),
				Sel:  ts.Ident(dstr),
			}, nil
		}
	default:
		return nil, valError(v, "unknown selector subject type %T, cannot translate", dvals[0].Source())
	}

	return nil, nil
}

// referenceValueAs returns the string that should be used to create a Typescript
// reference to the given struct, if a reference is allowable.
//
// References are only permitted to other Values with an @cuetsy(kind)
// attribute. The variadic parameter determines which kinds will be treated as
// permissible. By default, all kinds are permitted.
//
// An nil expr indicates a reference is not allowable, including the case
// that the provided Value is not actually a reference. A non-nil error
// indicates a deeper problem.
func referenceValueAs(v cue.Value, kinds ...tsKind) (ts.Expr, error) {
	op, dvals := v.Expr()

	// FIXME compensate for attribute-applying call to Unify() on incoming Value
	if op == cue.AndOp {
		v = dvals[0]
		op, dvals = dvals[0].Expr()
	}

	// References are primarily identified by their hallmark SelectorOp.
	if op != cue.SelectorOp {
		return nil, nil
	}

	// Have to do attribute checks on the referenced field itself, so deref
	deref := cue.Dereference(v)
	dstr, _ := dvals[1].String()

	// FIXME It's horrifying, teasing out the type of selector kinds this way. *Horrifying*.
	switch dvals[0].Source().(type) {
	case nil:
		// A nil subject means an unqualified selector (no "."
		// literal).  This can only possibly be a reference to some
		// sibling or parent of the top-level Value being generated.
		// (We can't do cycle detection with the meager tools
		// exported in cuelang.org/go/cue, so all we have for the
		// parent case is hopium.)
		if _, ok := dvals[1].Source().(*ast.Ident); ok && targetsKind(deref, kinds...) {
			return ts.Ident(dstr), nil
		}
	case *ast.SelectorExpr:
		// panic("case 2")
		if targetsKind(deref, kinds...) {
			return ts.Ident(dstr), nil
		}
	case *ast.Ident:
		// panic("case 3")
		if targetsKind(deref, kinds...) {
			str, ok := dvals[0].Source().(fmt.Stringer)
			if !ok {
				panic("expected dvals[0].Source() to implement String()")
			}

			return tsast.SelectorExpr{
				Expr: ts.Ident(str.String()),
				Sel:  ts.Ident(dstr),
			}, nil
		}
	default:
		return nil, valError(v, "unknown selector subject type %T, cannot translate", dvals[0].Source())
	}

	return nil, nil
}
