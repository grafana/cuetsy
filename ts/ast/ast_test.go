package ast_test

import (
	"testing"

	"github.com/grafana/cuetsy/ts/ast"
	"github.com/matryer/is"
)

func ident(s string) ast.Ident {
	return ast.Ident{Name: s}
}

func str(s string) ast.Str {
	return ast.Str{Value: s}
}

func TestSelectorExpr(t *testing.T) {
	is := is.New(t)

	expr := ast.SelectorExpr{
		Expr: ident("foo"),
		Sel:  ident("bar"),
	}
	is.Equal("foo.bar", expr.String())

	expr = ast.SelectorExpr{
		Expr: ast.SelectorExpr{
			Expr: ident("foo"),
			Sel:  ident("bar"),
		},
		Sel: ident("baz"),
	}
	is.Equal("foo.bar.baz", expr.String())
}

func TestIndexExpr(t *testing.T) {
	is := is.New(t)

	expr := ast.IndexExpr{
		Expr:  ident("foo"),
		Index: ast.Num{N: 3},
	}
	is.Equal(expr.String(), "foo[3]")
}

func TestNum(t *testing.T) {
	is := is.New(t)

	is.Equal("0", ast.Num{N: 0}.String())
	is.Equal("-12", ast.Num{N: -12}.String())
	is.Equal("8", ast.Num{N: 8}.String())

	is.Equal("0.12", ast.Num{N: 0.12}.String())
	is.Equal("312", ast.Num{N: 3.12e2}.String())
	is.Equal("3.120000e+02", ast.Num{N: 3.12e2, Fmt: "%e"}.String())
}

func TestAssignExpr(t *testing.T) {
	is := is.New(t)

	expr := ast.AssignExpr{
		Name:  ident("foo"),
		Value: ast.Num{N: 4},
	}
	is.Equal("foo = 4", expr.String())
}

func TestKeyValueExpr(t *testing.T) {
	is := is.New(t)

	expr := ast.KeyValueExpr{
		Key:   ident("foo"),
		Value: str("bar"),
	}
	is.Equal("foo: 'bar'", expr.String())

	expr = ast.KeyValueExpr{
		Key: ast.IndexExpr{
			Expr:  ident(""),
			Index: str("bar"),
		},
		Value: str("baz"),
	}
	is.Equal("['bar']: 'baz'", expr.String())
}

func TestEnumType(t *testing.T) {
	is := is.New(t)

	T := ast.EnumType{
		Elems: []ast.Expr{},
	}
	is.Equal("{}", T.String())

	T = ast.EnumType{
		Elems: []ast.Expr{
			ast.AssignExpr{Name: ident("foo"), Value: ast.Num{N: 1}},
			ident("bar"),
			ident("baz"),
		},
	}
	want := `{
  foo = 1,
  bar,
  baz,
}`
	is.Equal(want, T.String())
}

func TestIndentation(t *testing.T) {
	is := is.New(t)

	kv1 := kv(ident("foo"), ident("string"))
	obj1 := obj(
		kv(ident("astring"), ident("string")),
		kv(ident("anum"), ident("number")),
	)

	OT := obj(kv1, kv(ident("alist"), ast.ListExpr{obj1}))
	want := `{
  foo: string,
  alist: Array<{
    astring: string,
    anum: number,
  }>,
}`
	is.Equal(want, OT.String())

	IT := ast.InterfaceType{
		Elems: []ast.KeyValueExpr{
			kv1,
			kv(ident("alist"), ast.ListExpr{obj1}),
		},
	}

	want = `{
  foo: string;
  alist: Array<{
    astring: string;
    anum: number;
  }>;
}`
	is.Equal(want, IT.String())
}

func obj(kv ...ast.KeyValueExpr) ast.ObjectLit {
	return ast.ObjectLit{
		Elems: kv,
	}
}

func kv(k, v ast.Expr) ast.KeyValueExpr {
	return ast.KeyValueExpr{
		Key:   k,
		Value: v,
	}
}
