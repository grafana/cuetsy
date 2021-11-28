package ts

import (
	"fmt"
	"runtime"

	"github.com/grafana/cuetsy/ts/ast"
)

type (
	File = ast.File
	Node = ast.Node
)

func Ident(name string) ast.Ident {
	return ast.Ident{Name: name}
}

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

func Export(decl ast.Decl) ast.Node {
	return ast.ExportStmt{Decl: decl}
}

func Raw(data string) ast.Raw {
	pc, file, no, ok := runtime.Caller(1)
	details := runtime.FuncForPC(pc)
	if ok && details != nil {
		fmt.Printf("fix: ts.Raw used by %s at %s#%d\n", details.Name(), file, no)
	}

	return ast.Raw{Data: data}
}
