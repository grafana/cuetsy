package ts

import "github.com/grafana/cuetsy/ts/ast"

type (
	File = ast.File
	Node = ast.Node
)

func Union(elems ...ast.Expr) ast.Expr {
	switch len(elems) {
	case 0:
		return nil
	case 1:
		return elems[0]
	}

	var U ast.Expr = elems[0]
	for _, e := range elems[1:] {
		U = ast.BinaryExpr{
			Op: "|",
			X:  U,
			Y:  e,
		}
	}

	return U
}
