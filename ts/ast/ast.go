package ast

import (
	"fmt"
	"strings"
)

const Indent = "  "

type File struct {
	Nodes []Node
}

func (f File) String() string {
	var b strings.Builder

	for _, n := range f.Nodes {
		b.WriteString(n.String())
		b.WriteString("\n")
	}

	return b.String()
}

type Node interface {
	fmt.Stringer
}

type Expr interface {
	Node
	expr()
}

var (
	_ Expr = SelectorExpr{}
	_ Expr = IndexExpr{}
	_ Expr = Num{}
)

type Raw struct {
	Data string
}

func (r Raw) expr() {}
func (r Raw) String() string {
	return r.Data
}

type Ident struct {
	Name string
}

func (i Ident) expr() {}
func (i Ident) String() string {
	return i.Name
}

func None() Expr {
	return Ident{}
}

type SelectorExpr struct {
	Expr Expr
	Sel  Ident
}

func (s SelectorExpr) expr() {}
func (s SelectorExpr) String() string {
	return fmt.Sprintf("%s.%s", s.Expr, s.Sel)
}

type IndexExpr struct {
	Expr  Expr
	Index Expr
}

func (i IndexExpr) expr() {}
func (i IndexExpr) String() string {
	return fmt.Sprintf("%s[%s]", i.Expr, i.Index)
}

type AssignExpr struct {
	Name  Ident
	Value Expr
}

func (a AssignExpr) expr() {}
func (a AssignExpr) String() string {
	return fmt.Sprintf("%s = %s", a.Name, a.Value)
}

type KeyValueExpr struct {
	Key   Expr
	Value Expr
}

func (k KeyValueExpr) expr() {}
func (k KeyValueExpr) String() string {
	return fmt.Sprintf("%s: %s", k.Key, k.Value)
}

type UnaryExpr struct {
	Op   string // operator
	Expr Expr   // operand
}

func (u UnaryExpr) expr() {}
func (u UnaryExpr) String() string {
	return u.Op + u.Expr.String()
}

type Num struct {
	N   interface{}
	Fmt string
}

func (n Num) expr() {}
func (n Num) String() string {
	if n.Fmt == "" {
		return fmt.Sprintf("%v", n.N)
	}
	return fmt.Sprintf(n.Fmt, n.N)
}

type Str struct {
	Value string
}

func (s Str) expr() {}
func (s Str) String() string {
	return fmt.Sprintf(`'%s'`, s.Value)
}

type Type interface {
	Node
	typeName() string
}

var (
	_ Type = EnumType{}
	_ Type = InterfaceType{}
)

type TypeDecl struct {
	Name Ident
	Type Type
}

func (t TypeDecl) String() string {
	return fmt.Sprintf("%s %s %s", t.Type.typeName(), t.Name, t.Type)
}

type EnumType struct {
	Elems []Expr
}

func (e EnumType) typeName() string { return "enum" }
func (e EnumType) String() string {
	var b strings.Builder
	b.WriteString("{")
	if len(e.Elems) > 0 {
		b.WriteString("\n")
	}
	for _, e := range e.Elems {
		b.WriteString(Indent)
		b.WriteString(e.String())
		b.WriteString(",\n")
	}
	b.WriteString("}")
	return b.String()
}

type InterfaceType struct {
	Elems []KeyValueExpr
}

func (i InterfaceType) typeName() string { return "interface" }
func (i InterfaceType) String() string {
	var b strings.Builder
	b.WriteString("{\n")
	for _, e := range i.Elems {
		b.WriteString(Indent)
		b.WriteString(e.String())
		b.WriteString("\n")
	}
	b.WriteString("}")
	return b.String()
}
