package cuetsy

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
)

var jf io.Writer = ioutil.Discard

func init() {
	// prepdump()
}

type dumpFlag uint64

const (
	dumpErrs dumpFlag = 1 << iota
	dumpRef
	dumpAttrs
	dumpSource
	dumpKind
	dumpIncompleteKind
	dumpExpr
	dumpBool
	dumpBytes
	dumpDefault
	dumpDocs
	dumpErr
	dumpEval
	dumpExists
	dumpStruct
	dumpString
	dumpInt
	dumpInt64
	dumpUint64
	dumpFloat64
	dumpIsClosed
	dumpIsConcrete
	dumpLabel
	dumpList
	dumpNull
	dumpPath
)

// Helper masks for common exploration patterns
const (
	hdumpVals = dumpString | dumpInt | dumpInt64 | dumpUint64 | dumpFloat64 | dumpStruct
	hdumpRefs = dumpExpr | dumpRef | dumpStruct | dumpPath | dumpString | dumpAttrs
	hdumpTyp  = dumpExpr | dumpKind | dumpIncompleteKind
)

func prepdump() {
	var err error
	jf, err = os.OpenFile("dump.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)

	if err != nil {
		panic(err)
	}
}

// Exercises all read-only funcs on a cue.Value.
//
// For great grokking, because the underlying datastructures are too complex to
// grasp directly.
func dumpJSON(name string, v cue.Value, flag dumpFlag) {
	whole := map[string]interface{}{
		name: assembleValues(v, flag, 0),
	}
	b, err := json.MarshalIndent(whole, "", "  ")
	if err != nil {
		panic(err)
	}

	fmt.Fprint(jf, string(b))
}

func assembleValues(v cue.Value, flags dumpFlag, depth int) (ret map[string]interface{}) {
	ret = make(map[string]interface{})
	r := func(iv cue.Value) map[string]interface{} {
		return assembleValues(iv, flags, depth+1)
	}

	fl := func(flag dumpFlag) bool {
		return flag&flags == flag
	}

	if depth > 4 {
		ret["STOP"] = "RECURSION"
		return
	}

	doAttrs := func(v cue.Value) (attrs []string) {
		if fl(dumpAttrs) {
			for _, attr := range v.Attributes(cue.ValueAttr) {
				attrs = append(attrs, fmt.Sprint(attr))
			}
		}
		return
	}

	if attrs := doAttrs(v); len(attrs) > 0 {
		ret["Attrs()"] = attrs
	}

	ref := func(v cue.Value) map[string]interface{} {
		ret := make(map[string]interface{})
		dv := cue.Dereference(v)
		ret["eq"] = dv.Equals(v)
		// ret["Dereference()"] = r(dv)

		if dattrs := doAttrs(dv); len(dattrs) > 0 {
			ret["Attrs()"] = dattrs
		}

		root, path := v.ReferencePath()
		if root.Exists() {
			ret["ReferencePath()"] = path.String()
		}
		return ret
	}

	if fl(dumpRef) {
		ret["ref"] = ref(v)
	}

	if src := v.Source(); fl(dumpSource) && src != nil {
		if isrc, ok := src.(*ast.Ident); ok {
			ret["Source()"] = []string{fmt.Sprint(src), fmt.Sprintf("%T", isrc), fmt.Sprint(isrc.Scope), fmt.Sprint(isrc.Node)}
		} else {
			ret["Source()"] = []string{fmt.Sprint(src), fmt.Sprintf("%T", src)}
		}
	}

	if fl(dumpBool) {
		if br, err := v.Bool(); err == nil {
			ret["Bool()"] = br
		} else if fl(dumpErrs) {
			ret["ERR Bool()"] = err
		}
	}

	if fl(dumpBytes) {
		if by, err := v.Bytes(); err == nil {
			ret["Bytes()"] = string(by)
		} else if fl(dumpErrs) {
			ret["ERR Bytes()"] = err
		}
	}

	// skip Decimal (internal only)

	if fl(dumpDefault) {
		if def, exists := v.Default(); exists {
			ret["Default()"] = r(def)
		}
	}

	if fl(dumpDocs) {
		if docs := v.Doc(); len(docs) > 0 {
			var docsl []string
			for _, d := range docs {
				docsl = append(docsl, fmt.Sprint(d))
			}
			ret["Docs()"] = docsl
		}
	}

	if fl(dumpErr) {

		if err := v.Err(); err != nil {
			ret["Err()"] = err
		}
	}

	if fl(dumpEval) {
		ret["Eval()"] = r(v.Eval())
	}

	if fl(dumpExists) {
		ret["Exists()"] = v.Exists()
	}

	// Skip Fields(), walking is up to the caller

	if fl(dumpFloat64) {
		if v2, err := v.Float64(); err == nil {
			ret["Float64()"] = v2
		} else if fl(dumpErrs) {
			ret["ERR Float64()"] = err
		}
	}

	if fl(dumpInt) {
		if v2, err := v.Int(nil); err == nil {
			ret["Int()"] = v2
		} else if fl(dumpErrs) {
			ret["ERR Int()"] = err
		}
	}

	if fl(dumpInt64) {
		if v2, err := v.Int64(); err == nil {
			ret["Int64()"] = v2
		} else if fl(dumpErrs) {
			ret["ERR Int64()"] = err
		}
	}

	if fl(dumpIsClosed) {
		ret["IsClosed()"] = v.IsClosed()
	}
	if fl(dumpIsConcrete) {
		ret["IsConcrete()"] = v.IsConcrete()
	}
	if fl(dumpKind) {
		ret["Kind()"] = fmt.Sprint(v.Kind())
	}

	if fl(dumpLabel) {
		if label, exists := v.Label(); exists {
			ret["Label()"] = label
		}
	}

	// Skipping Len. If the return is just a number, why is it a Value?

	if fl(dumpList) {
		if _, err := v.List(); err == nil {
			ret["List()"] = "returns iter"
		} else if fl(dumpErrs) {
			ret["ERR List()"] = err
		}
	}

	if fl(dumpNull) {
		if err := v.Null(); err == nil {
			ret["Null()"] = "yup it's null"
		} else if fl(dumpErrs) {
			ret["ERR Null()"] = err
		}
	}

	ret["Path()"] = fmt.Sprint(v.Path())

	// Skip Pos()
	// Skip Reader()

	if fl(dumpString) {
		if v2, err := v.String(); err == nil {
			ret["String()"] = v2
		} else if fl(dumpErrs) {
			ret["ERR String()"] = err
		}
	}

	if fl(dumpStruct) {
		if strc, err := v.Struct(); err == nil {
			sub := make(map[string]map[string]interface{})
			var fld []string
			for i := 0; i < strc.Len(); i++ {
				sk, sv := strc.At(i)
				sub[sk] = r(sv)
				fld = append(fld, sk)
			}
			// ret["Struct() fields"] = sub
			ret["Struct() fields"] = fld
		} else if fl(dumpErrs) {
			ret["ERR Struct()"] = err
		}
	}

	// ret["Syntax()"] = v.Syntax(cue.All())

	if fl(dumpUint64) {
		if v2, err := v.Uint64(); err == nil {
			ret["Uint64()"] = v2
		} else if fl(dumpErrs) {
			ret["ERR Uint64()"] = err
		}
	}

	// Skip Validate()

	if fl(dumpExpr) {
		op, vals := v.Expr()
		if op != cue.NoOp {
			var exprvals []map[string]interface{}
			for _, val := range vals {
				if !v.Equals(val) {
					exprvals = append(exprvals, r(val))
				} else {
					exprvals = append(exprvals, map[string]interface{}{"(self)": "Equal()s parent"})
				}
			}
			if len(exprvals) > 0 {
				ret["Expr()"] = map[string]interface{}{
					"Op":    op.String(),
					"Parts": exprvals,
				}
			}
		}
	}

	return
}
