package cuetsy

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
)

func testAnalyzeReference(t *testing.T) {
	str := `
import "list"

node1: string @cuetsy(kind="type")
node2: string | *int @cuetsy(kind="type")
node3: *"foo" | "bar" | "baz" @cuetsy(kind="enum", memberNames="Foo|Bar|Baz")
node4: node3 | node2
node5: {
	inner: string
	n3: node3
} @cuetsy(kind="interface")

// zoom: {...} // if we unify with open struct, all the Source calls become nil :(
zoom: {}
zow: {}

ref: zoom & {
	bigref: node4 | node5.n3 | *"foo"
	// zow
	// n1: node1
	// n2: node2 | node3
	// n3: node2 | node3 | *"bar"
	// n3has: node3 | *"bar"
	// n3new: node3 | *"bix"
	// n3l: "foo" | *"bar" | "baz"
	// listlit: [("foo" | "bar" | "boink"), ...("foo" | *"bar" | "baz")]
	// lst: [...node3] | *listlit | node5
} @cuetsy(kind="interface")
`

	ctx := cuecontext.New()
	val := ctx.CompileString(str)
	// fmt.Println(val.LookupPath(cue.ParsePath("node1")).Pos())

	ref := val.LookupPath(cue.ParsePath("ref"))
	iter, err := ref.Fields(cue.Optional(true))
	if err != nil {
		t.Fatal(err)
	}

	for iter.Next() {
		s, v := iter.Selector(), iter.Value()
		dv := cue.Dereference(v)
		op, dvals := v.Expr()
		_, _, _, _, _ = s, v, dv, op, dvals

		fmt.Println(s, "--", v)
		et := exprTree(v)
		fmt.Println(et)
		// printSplit(v, cue.OrOp)
		// printSource(appendSplit(nil, cue.OrOp, v)...)
		// printSplit(v, cue.AndOp)

		// fmt.Println("CONTAINSCUETSYREF", containsCuetsyReference(v))

		// printSource(v, dv, dvals[0])
		// printAttrs(v, dv, dvals[0])

		// fmt.Printf("EXPR: %#v %s, %v\n", op, dvals, len(dvals))
	}

	// op, dvals := ref.LookupPath(cue.ParsePath("n3l")).Expr()
	// fmt.Printf("EXPR: %#v %s\n", op, dvals)
}

func printSplit(v cue.Value, op cue.Op) {
	a := appendSplit(nil, op, v)
	for _, splitv := range a {
		_, r := splitv.Reference()
		fmt.Printf("\t%v: %+v\n", op, splitv)
		if len(r) > 0 {
			fmt.Printf("\t\t%v\n", strings.Join(r, "."))
		}
	}
}

// func checkDefault(v cue.Value) {
// 	def, has := v.Default()
// 	if !has {
// 		return
// 	}
// 	fmt.Println(def)
// 	if containsReference(def) {
// 		fmt.Println("has a ref")
// 	} else {
// 		fmt.Println("already deref'd")
// 	}
// 	// fmt.Println("--", s)
// 	flat := flatten(def)
//
// 	fmt.Println("DEFLEN", len(flat))
// 	for _, dv := range flat {
// 		rv, refpath := dv.ReferencePath()
// 		fmt.Println(dv.Path(), refpath, dv)
// 		if len(refpath.Selectors()) > 0 {
// 			printAttrs(rv.LookupPath(refpath))
// 		}
// 	}
// }

func printSource(vl ...cue.Value) {
	for _, v := range vl {
		s := v.Source()
		fmt.Printf("\t%v, %T, %v\n", v.Path(), s, s)
		if f, ok := s.(*ast.Field); ok {
			fmt.Printf("\t\tFIELDXPR %T %v\n", f.Value, f.Value)
		}
	}
}

func printAttrs(vl ...cue.Value) {
	for _, v := range vl {
		fmt.Println(v.Attributes(cue.ValueAttr))
	}
}

var tt = []struct {
	name        string
	cuein       string
	expectProps []listProps
}{
	{
		name:  "empty",
		cuein: `[]`,
		expectProps: []listProps{
			{
				emptyDefault: true,
			},
		},
	},
	{
		name:  "simpleopen",
		cuein: `[...string]`,
		expectProps: []listProps{
			{
				isOpen:          true,
				emptyDefault:    true,
				bottomKinded:    true,
				argBottomKinded: true,
			},
		},
	},
	{
		name:  "openemptydefault",
		cuein: `[...string] | *[]`,
		expectProps: []listProps{
			{
				isOpen:           true,
				emptyDefault:     true,
				differentDefault: true,
				bottomKinded:     true,
				argBottomKinded:  true,
			},
		},
	},
	{
		name:  "revopenemptydefault",
		cuein: `*[] | [...string]`,
		expectProps: []listProps{
			{
				isOpen:           true,
				emptyDefault:     true,
				differentDefault: true,
				bottomKinded:     true,
				argBottomKinded:  true,
			},
		},
	},
	{
		name:  "oneplusopen",
		cuein: `[string, ...string]`,
		expectProps: []listProps{
			{
				isOpen:          true,
				bottomKinded:    true,
				argBottomKinded: true,
				// differentDefault: true,
			},
		},
	},
	// {
	// 	name:  "listminopen",
	// 	cuein: `[...string] & list.MinItems(1)`,
	// 	expectProps: []listProps{
	// 		{
	// 			isOpen:           true,
	// 			noDefault:       true,
	// 			differentDefault: true,
	// 			emptyDefault:     true,
	// 		},
	// 	},
	// },
	{
		name:  "simpleclosed",
		cuein: `[string]`,
		expectProps: []listProps{
			{
				emptyDefault: false,
			},
		},
	},
	{
		name:  "concrete1",
		cuein: `["foo"]`,
		expectProps: []listProps{
			{},
		},
	},
	{
		name:  "concrete2",
		cuein: `["foo", "bar"]`,
		expectProps: []listProps{
			{
				divergentTypes: true, // ugh, it should really be divergentKinds
			},
		},
	},
	{
		name:  "concretemultitype",
		cuein: `["foo", 2]`,
		expectProps: []listProps{
			{
				divergentTypes: true,
			},
		},
	},
	{
		name:  "simpledisj",
		cuein: `[...string] | [...int]`,
		expectProps: []listProps{
			{
				differentDefault: true,
				emptyDefault:     true,
				isOpen:           true,
			},
			{
				differentDefault: true,
				emptyDefault:     true,
				isOpen:           true,
				bottomKinded:     true,
				argBottomKinded:  true,
			},
			{
				differentDefault: true,
				emptyDefault:     true,
				isOpen:           true,
				bottomKinded:     true,
				argBottomKinded:  true,
			},
		},
	},
	{
		name:  "simpledefault",
		cuein: `[...string] | *["foo"]`,
		expectProps: []listProps{
			{
				differentDefault: true,
				isOpen:           true,
				bottomKinded:     true,
				argBottomKinded:  true,
			},
		},
	},
	{
		name:  "multitypedefault",
		cuein: `[...string] | *[int]`,
		expectProps: []listProps{
			{
				isOpen:       true,
				bottomKinded: true,
			},
		},
	},
}

// Two paths for a value
// - it does not contain a reference (easy)
// - it contains a reference, at some level (but NOT descending into structs). question: do we dereference?
//   - maybe it's just a straight reference, no frills. Then, decide whether to deref or not based on attr of target
//   - could be a disjunction with references on some branches. Independent of whether there are default branch(es)
//     - there may be a branch with a default
//     - o

// What are the kinds of references?
// - Simple, direct references to other types (with|without @cuetsy)
// - Direct references that add a default value
// - Disjuncts over other direct references

func testAnalyzeList(t *testing.T) {
	ctx := cuecontext.New()

	boolf := func(t *testing.T, field string, expect bool) {
		t.Helper()

		if expect {
			t.Logf("%s: should be true", field)
		} else {
			t.Logf("%s: should be false", field)
		}

		t.Fail()
	}

	for _, test := range tt {
		item := test
		t.Run(item.name, func(t *testing.T) {
			lfs := analyzeList(ctx.CompileString(item.cuein))
			if len(item.expectProps) != len(lfs) {
				t.Fatalf("expected %v discrete list items, got %v", len(item.expectProps), len(lfs))
			}
			for i, lfield := range lfs {
				lf := lfield
				l := lf.props
				props := item.expectProps[i]
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					if props.isOpen != l.isOpen {
						boolf(t, "isOpen", props.isOpen)
					}
					if props.divergentTypes != l.divergentTypes {
						boolf(t, "divergentTypes", props.divergentTypes)
					}
					if props.noDefault != l.noDefault {
						boolf(t, "noDefault", props.noDefault)
					}
					if props.differentDefault != l.differentDefault {
						boolf(t, "differentDefault", props.differentDefault)
					}
					if props.emptyDefault != l.emptyDefault {
						boolf(t, "emptyDefault", props.emptyDefault)
					}
					if props.bottomKinded != l.bottomKinded {
						boolf(t, "bottomKinded", props.bottomKinded)
					}
					if props.argBottomKinded != l.argBottomKinded {
						boolf(t, "argBottomKinded", props.argBottomKinded)
					}

					if t.Failed() {
						t.Logf("%s\n", lf)
						// defv, _ := lf.v.Default()
						// fmt.Println(dumpsynP(defv))
					}
				})
			}
		})
	}

}

func testGroupList(t *testing.T) {
	ctx := cuecontext.New()
	var all []*listField
	for _, test := range tt {
		lfs := analyzeList(ctx.CompileString(test.cuein))
		all = append(all, lfs...)
	}

	has, not := groupBy(all, func(props listProps) bool {
		return props.bottomKinded
	})
	printgrouping("bottomKinded", has, not)

	has, not = groupBy(all, func(props listProps) bool {
		return props.argBottomKinded
	})
	printgrouping("argBottomKinded", has, not)

	has, not = groupBy(all, func(props listProps) bool {
		return props.isOpen
	})
	printgrouping("isOpen", has, not)

	has, not = groupBy(all, func(props listProps) bool {
		return props.divergentTypes
	})
	printgrouping("divergentTypes", has, not)

	has, not = groupBy(all, func(props listProps) bool {
		return props.emptyDefault
	})
	printgrouping("emptyDefault", has, not)

	has, not = groupBy(all, func(props listProps) bool {
		return props.differentDefault
	})
	printgrouping("differentDefault", has, not)
}

func printgrouping(field string, has, not []*listField) {
	if len(has) > 0 {
		fmt.Println(field, "is TRUE:")
		for _, lf := range has {
			fmt.Println("\t", dumpsynP(lf.v))
		}
	}

	if len(not) > 0 {
		fmt.Println(field, "is FALSE:")
		for _, lf := range not {
			fmt.Println("\t", dumpsynP(lf.v))
		}
	}
	fmt.Println()
}

// whyyyy doesn't this work?
func testExtractComment(t *testing.T) {
	str := `
// comment floater

withattr: string @cuetsy(kind="type") // inline withattr
noattr: int32 // inline noattr
`

	ctx := cuecontext.New()
	val := ctx.CompileString(str)
	printcgs("base", val.Doc())

	iter, err := val.Fields(cue.Optional(true))
	if err != nil {
		t.Fatal(err)
	}

	for iter.Next() {
		printcgs(iter.Selector().String(), iter.Value().Doc())
	}
}

func printcgs(pre string, cgs []*ast.CommentGroup) {
	fmt.Println(pre)
	for _, cg := range cgs {
		fmt.Println(cg.Text())
	}
	fmt.Println()
}

func testRelDisjunct(t *testing.T) {
	str := `disj: "foo" | "bar"
check: {
  member: "foo"
  notmember: "baz"
  subsumer: string
  wider: "foo" | "bar" | "baz"
  partialOverlap: "foo" | "baz"
}
`
	ctx := cuecontext.New()
	val := ctx.CompileString(str)

	dv := val.LookupPath(cue.ParsePath("disj"))
	iter, err := val.LookupPath(cue.ParsePath("check")).Fields()
	if err != nil {
		t.Fatal(err)
	}

	for iter.Next() {
		checkv := iter.Value()
		t.Run(iter.Selector().String(), func(t *testing.T) {
			// t.Run(valForTestName(iter.Value()), func(t *testing.T) {
			t.Run("subsume", func(t *testing.T) {
				subt := func(a, b cue.Value) func(t *testing.T) {
					return func(t *testing.T) {
						t.Logf("(%#v)⊑(%#v)", a, b)
						logif(t, "NOOPTS:", a.Subsume(b))
						logif(t, "RAW:", a.Subsume(b, cue.Raw()))
						logif(t, "CONCRETE:", a.Subsume(b, cue.Concrete(true)))
						logif(t, "RAW|CONCRETE:", a.Subsume(b, cue.Raw(), cue.Concrete(true)))
						logif(t, "FINAL:", a.Subsume(b, cue.Final()))
						logif(t, "RAW|FINAL:", a.Subsume(b, cue.Raw(), cue.Final()))
						logif(t, "CONCRETE|FINAL:", a.Subsume(b, cue.Concrete(true), cue.Final()))
						logif(t, "RAW|CONCRETE|FINAL:", a.Subsume(b, cue.Raw(), cue.Concrete(true), cue.Final()))
					}
				}
				t.Run("d(c)", subt(dv, checkv))
				t.Run("c(d)", subt(checkv, dv))
			})
		})
		t.Run("concrete", func(t *testing.T) {
			t.Logf("%#v is concrete: %v", checkv, checkv.IsConcrete())
		})
	}
}

func logif(t *testing.T, fmt string, err error) {
	t.Helper()
	if err != nil {
		t.Log(fmt, err)
	}
}

func valForTestName(v cue.Value) string {
	return strings.Replace(fmt.Sprintf("%#v", v), " ", "", -1)
}
