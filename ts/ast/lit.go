package ast

import "strings"

type ObjectLit struct {
	Elems []KeyValueExpr

	eol EOL
	lvl int
}

func (o ObjectLit) expr() {}
func (o ObjectLit) String() string {
	if o.lvl == 0 {
		o.lvl = 1
	}

	eol := string(o.eol)
	if eol == "" {
		eol = string(EOLComma)
	}
	eol += "\n"

	var b strings.Builder
	write := b.WriteString
	indent := func(n int) {
		write(strings.Repeat(Indent, n))
	}

	write("{\n")
	for _, e := range o.Elems {
		if oo, ok := e.Value.(ObjectLit); ok {
			oo.eol = o.eol
			oo.lvl = o.lvl + 1
			e.Value = oo
		}

		indent(o.lvl)
		write(e.String())
		write(eol)
	}

	indent(o.lvl - 1)
	write("}")

	return b.String()
}

type ListLit struct {
	Elems []Expr
}

func (l ListLit) expr() {}
func (l ListLit) String() string {
	strs := make([]string, len(l.Elems))
	for i, e := range l.Elems {
		strs[i] = e.String()
	}
	return string(SquareBrack[0]) + strings.Join(strs, ", ") + string(SquareBrack[1])
}

// TODO: combine InterfaceType, EnumType, ListLit and ObjectLit rendering into below
// type CompositeLit struct {}
