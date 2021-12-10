package ast

import "strings"

type DestrLit struct {
	Brack  Brack
	Idents []Ident
}

func (d DestrLit) ident() {}
func (d DestrLit) String() string {
	idStrs := make([]string, len(d.Idents))
	for i, d := range d.Idents {
		idStrs[i] = d.String()
	}

	return string(d.Brack[0]) + strings.Join(idStrs, ", ") + string(d.Brack[1])
}

type ObjectLit struct {
	Elems []KeyValueExpr
}

func (o ObjectLit) expr() {}
func (o ObjectLit) String() string {
	var b strings.Builder
	b.WriteString("{\n")
	for _, e := range o.Elems {
		b.WriteString(Indent)
		b.WriteString(e.String())
		b.WriteString(",\n")
	}
	b.WriteString("}")
	return b.String()
}

// TODO: combine InterfaceType, EnumType and ObjectLit rendering into below
// type CompositeLit struct {}
