package ast

import (
	"strings"
)

type ObjectLit struct {
	Elems  []KeyValueExpr
	IsType bool

	eol EOL
	lvl int
}

func (o ObjectLit) expr() {}
func (o ObjectLit) String() string {
	if len(o.eol) == 0 {
		return o.innerString(EOLComma, o.lvl)
	}
	return o.innerString(o.eol, o.lvl)
}

func (o ObjectLit) innerString(aeol EOL, lvl int) string {
	lvl++
	eol := string(aeol) + "\n"

	var b strings.Builder
	write := b.WriteString
	indent := func(n int) {
		write(strings.Repeat(Indent, n))
	}

	if len(o.Elems) == 0 {
		if o.IsType {
			write("Record<string, unknown>")
		} else {
			write("{}")
		}
		return b.String()
	}

	write("{\n")
	for _, e := range o.Elems {
		indent(lvl)
		write(innerString(aeol, lvl, e))
		write(eol)
	}

	indent(lvl - 1)
	write("}")

	return b.String()
}

type ListLit struct {
	Elems []Expr
}

func innerString(eol EOL, lvl int, e Expr) string {
	if x, ok := e.(innerStringer); ok {
		return x.innerString(eol, lvl)
	}
	return e.String()
}

func (l ListLit) expr() {}
func (l ListLit) String() string {
	return l.innerString(EOLComma, 0)
}

func (l ListLit) innerString(eol EOL, lvl int) string {
	strs := make([]string, len(l.Elems))
	for i, e := range l.Elems {
		strs[i] = innerString(eol, lvl, e)
	}
	return string(SquareBrack[0]) + strings.Join(strs, ", ") + string(SquareBrack[1])
}

// TODO: combine InterfaceType, EnumType, ListLit and ObjectLit rendering into below
// type CompositeLit struct {}
