package ast_test

import (
	"testing"

	"github.com/grafana/cuetsy/ts/ast"
	"github.com/matryer/is"
)

func TestSelectorExpr(t *testing.T) {
	is := is.New(t)

	expr := ast.SelectorExpr{
		Expr: ast.Ident{"foo"},
		Sel:  ast.Ident{"bar"},
	}
	is.Equal("foo.bar", expr.String())

	expr = ast.SelectorExpr{
		Expr: ast.SelectorExpr{
			Expr: ast.Ident{"foo"},
			Sel:  ast.Ident{"bar"},
		},
		Sel: ast.Ident{"baz"},
	}
	is.Equal("foo.bar.baz", expr.String())
}

func TestIndexExpr(t *testing.T) {
	is := is.New(t)

	expr := ast.IndexExpr{
		Expr:  ast.Ident{"foo"},
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
		Name:  ast.Ident{"foo"},
		Value: ast.Num{N: 4},
	}
	is.Equal("foo = 4", expr.String())
}

func TestKeyValueExpr(t *testing.T) {
	is := is.New(t)

	expr := ast.KeyValueExpr{
		Key:   ast.Ident{"foo"},
		Value: ast.Str{"bar"},
	}
	is.Equal("foo: 'bar'", expr.String())

	expr = ast.KeyValueExpr{
		Key: ast.IndexExpr{
			Expr:  ast.Ident{""},
			Index: ast.Str{"bar"},
		},
		Value: ast.Str{"baz"},
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
			ast.AssignExpr{Name: ast.Ident{"foo"}, Value: ast.Num{N: 1}},
			ast.Ident{"bar"},
			ast.Ident{"baz"},
		},
	}
	want := `{
  foo = 1,
  bar,
  baz,
}`
	is.Equal(want, T.String())
}
