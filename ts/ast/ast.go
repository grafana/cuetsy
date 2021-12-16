package ast

import (
	"fmt"
	"strings"
)

const Indent = "  "

type Brack string

const (
	RoundBrack  Brack = "()"
	SquareBrack Brack = "[]"
	CurlyBrack  Brack = "{}"
)

type File struct {
	Imports []ImportSpec
	Nodes   []Decl
}

func (f File) String() string {
	var b strings.Builder

	for _, i := range f.Imports {
		b.WriteString(i.String())
		b.WriteString("\n\n")
	}

	for i, n := range f.Nodes {
		b.WriteString(n.String())

		if i+1 < len(f.Nodes) {
			b.WriteString("\n\n")
		}
	}
	b.WriteString("\n")

	return b.String()
}

type Node interface {
	fmt.Stringer
}

type Expr interface {
	Node
	expr()
}

type Decl interface {
	Node
	decl()
}

type Idents interface {
	Node
	ident()
}

var (
	_ Idents = Ident{}
	_ Idents = DestrLit{}
)

var (
	_ Expr = SelectorExpr{}
	_ Expr = IndexExpr{}
	_ Expr = Num{}
)

type Raw struct {
	Data string
}

func (r Raw) decl() {}
func (r Raw) expr() {}
func (r Raw) String() string {
	return r.Data
}

type Ident struct {
	Name string
	As   string
}

func (i Ident) ident() {}
func (i Ident) expr()  {}
func (i Ident) String() string {
	if i.As != "" {
		return fmt.Sprintf("%s as %s", i.Name, i.As)
	}
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

type BinaryExpr struct {
	Op   string
	X, Y Expr
}

func (b BinaryExpr) expr() {}
func (b BinaryExpr) String() string {
	return fmt.Sprintf("%s %s %s", b.X, b.Op, b.Y)
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

type VarDecl struct {
	Tok string

	Name  Idents
	Type  Ident
	Value Expr
}

func (v VarDecl) decl() {}
func (v VarDecl) String() string {
	tok := v.Tok
	if tok == "" {
		tok = "const"
	}
	return fmt.Sprintf("%s %s: %s = %s;", tok, v.Name, v.Type, v.Value)
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

func (t TypeDecl) decl() {}
func (t TypeDecl) String() string {
	return fmt.Sprintf("%s %s %s", t.Type.typeName(), t.Name, t.Type)
}

type BasicType struct {
	Expr Expr
}

func (b BasicType) typeName() string { return "type" }
func (b BasicType) String() string {
	return fmt.Sprintf("= %s;", b.Expr)
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
	Elems   []KeyValueExpr
	Extends []Ident
}

func (i InterfaceType) typeName() string { return "interface" }
func (i InterfaceType) String() string {
	var b strings.Builder
	if len(i.Extends) > 0 {
		b.WriteString("extends ")
		for i, s := range i.Extends {
			if i != 0 {
				b.WriteString(", ")
			}
			b.WriteString(s.Name)
		}
		b.WriteString(" ")
	}

	b.WriteString("{\n")
	for _, e := range i.Elems {
		b.WriteString(Indent)
		b.WriteString(e.String())
		b.WriteString(";\n")
	}
	b.WriteString("}")
	return b.String()
}

type ExportStmt struct {
	Decl Decl
}

func (e ExportStmt) decl() {}
func (e ExportStmt) String() string {
	return "export " + e.Decl.String()
}

type ImportSpec struct {
	From  Str
	Names Idents
}

func (i ImportSpec) String() string {
	return fmt.Sprintf("import %s from %s;", i.Names, i.From)
}
