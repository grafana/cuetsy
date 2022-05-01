package cuetsy

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"
)

func isReference(v cue.Value) bool {
	_, path := v.ReferencePath()
	if len(path.Selectors()) > 0 {
		return true
	}

	return false
}

func getKindFor(v cue.Value) (tsKind, error) {
	// Direct lookup of attributes with Attribute() seems broken-ish, so do our
	// own search as best we can, allowing ValueAttrs, which include both field
	// and decl attributes.
	// TODO write a unit test checking expected attribute output behavior to
	// protect this brittleness against regressions due to language changes
	var found bool
	var attr cue.Attribute
	for _, a := range v.Attributes(cue.ValueAttr) {
		if a.Name() == attrname {
			found = true
			attr = a
		}
	}
	if !found {
		return "", valError(v, "value has no \"@%s\" attribute", attrname)
	}

	tt, found, err := attr.Lookup(0, attrKind)
	if err != nil {
		return "", err
	}

	if !found {
		return "", valError(v, "no value for the %q key in @%s attribute", attrKind, attrname)
	}
	return tsKind(tt), nil
}

func getForceText(v cue.Value) string {
	var found bool
	var attr cue.Attribute
	for _, a := range v.Attributes(cue.ValueAttr) {
		if a.Name() == attrname {
			found = true
			attr = a
		}
	}
	if !found {
		return ""
	}

	ft, found, err := attr.Lookup(0, attrForceText)
	if err != nil || !found {
		return ""
	}

	return ft
}

func targetsAnyKind(v cue.Value) bool {
	return targetsKind(v)
}

func targetsKind(v cue.Value, kinds ...tsKind) bool {
	vkind, err := getKindFor(v)
	if err != nil {
		return false
	}

	if len(kinds) == 0 {
		kinds = allKinds[:]
	}
	for _, knd := range kinds {
		if vkind == knd {
			return true
		}
	}
	return false
}

// containsReference recursively flattens expressions within a Value to find all
// its constituent Values, and checks if any of those Values are references.
//
// It does NOT walk struct fields - only expression structures, as returned from Expr().
// Remember that Expr() _always_ drops values in default branches.
func containsReference(v cue.Value) bool {
	if isReference(v) {
		return true
	}
	for _, dv := range flatten(v) {
		if isReference(dv) {
			return true
		}
	}
	return false
}

// containsCuetsyReference does the same as containsReference, but returns true
// iff at least one referenced node passes the targetsKind predicate check
func containsCuetsyReference(v cue.Value, kinds ...tsKind) bool {
	if isReference(v) && targetsKind(cue.Dereference(v), kinds...) {
		return true
	}
	for _, dv := range flatten(v) {
		if isReference(dv) && targetsKind(cue.Dereference(dv), kinds...) {
			return true
		}
	}
	return false
}

type valuePredicate func(cue.Value) bool

type valuePredicates []valuePredicate

func (pl valuePredicates) And(v cue.Value) bool {
	for _, p := range pl {
		if !p(v) {
			return false
		}
	}
	return true
}

func (pl valuePredicates) Or(v cue.Value) bool {
	for _, p := range pl {
		if p(v) {
			return true
		}
	}
	return len(pl) == 0
}

func containsPred(v cue.Value, depth int, pl ...valuePredicate) bool {
	vpl := valuePredicates(pl)
	if vpl.And(v) {
		return true
	}
	// for _, v := range flatten(v) {
	// 	if vpl.And(v) {
	// 		return true
	// 	}
	// }
	if depth != 0 {
		op, args := v.Expr()
		_, has := v.Default()
		if op != cue.NoOp || has {
			for _, dv := range args {
				if containsPred(dv, depth-1, vpl...) {
					return true
				}
			}
		}
	}
	return false
}

func flatten(v cue.Value) []cue.Value {
	var all []cue.Value

	_, dvals := v.Expr()
	all = append(all, dvals...)
	if len(dvals) != 1 {
		for _, dv := range dvals {
			all = append(all, flatten(dv)...)
		}
	}
	return all
}

// appendSplit splits a cue.Value into the
func appendSplit(a []cue.Value, splitBy cue.Op, v cue.Value) []cue.Value {
	op, args := v.Expr()
	// dedup elements.
	k := 1
outer:
	for i := 1; i < len(args); i++ {
		for j := 0; j < k; j++ {
			if args[i].Subsume(args[j], cue.Raw()) == nil &&
				args[j].Subsume(args[i], cue.Raw()) == nil {
				continue outer
			}
		}
		args[k] = args[i]
		k++
	}
	args = args[:k]

	if op == cue.NoOp && len(args) == 1 {
		// TODO: this is to deal with default value removal. This may change
		a = append(a, args...)
	} else if op != splitBy {
		a = append(a, v)
	} else {
		for _, v := range args {
			a = appendSplit(a, splitBy, v)
		}
	}
	return a
}

func dumpsyn(v cue.Value) (string, error) {
	syn := v.Syntax(
		cue.Concrete(false), // allow incomplete values
		cue.Definitions(false),
		cue.Optional(true),
		cue.Attributes(true),
		cue.Docs(true),
		cue.ResolveReferences(false),
	)

	byt, err := format.Node(syn, format.Simplify(), format.TabIndent(true))
	return string(byt), err
}
