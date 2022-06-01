package ast

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const Indent = "  "

type Brack string

const (
	RoundBrack  Brack = "()"
	SquareBrack Brack = "[]"
	CurlyBrack  Brack = "{}"
)

type Quot string

const (
	SingleQuot Quot = `'`
	DoubleQuot Quot = `"`
	BTickQuot  Quot = "`"
)

type EOL string

const (
	EOLComma     EOL = `,`
	EOLSemicolon EOL = `;`
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

type innerStringer interface {
	innerString(eol EOL, lvl int) string
}

type Expr interface {
	Node
	expr()
}

type Decl interface {
	Node
	decl()
}

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

	// TODO: factor out into asStmt?
	As string
}

var identRegexp = regexp.MustCompile("^[a-zA-Z_$][0-9a-zA-Z_$]*$")

var ErrBadIdent = errors.New("typescript idents must contain only alphanumeric characters")

func (i Ident) ident() {}
func (i Ident) expr()  {}
func (i Ident) String() string {
	n := strings.Replace(i.Name, "#", "", -1)

	if i.As != "" {
		return fmt.Sprintf("%s as %s", n, i.As)
	}
	return n
}

func (i Ident) Validate() error {
	if !identRegexp.MatchString(i.Name) {
		return ErrBadIdent
	}
	if i.As != "" && !identRegexp.MatchString(i.Name) {
		return ErrBadIdent
	}
	return nil
}

func None() Expr {
	return Ident{}
}

type Idents []Ident

func (i Idents) Strings() []string {
	strs := make([]string, len(i))
	for i, id := range i {
		strs[i] = id.Name
	}
	return strs
}

type Names struct {
	Brack
	Idents
}

func (n Names) String() string {
	switch len(n.Idents) {
	case 0:
		panic("Names.Idents must not be empty")
	case 1:
		return n.Idents[0].String()
	}

	b := n.Brack
	if b == "" {
		b = CurlyBrack
	}

	return fmt.Sprintf("%c%s%c", b[0], strings.Join(n.Idents.Strings(), ","), b[1])
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
	return k.innerString(EOL(""), 0)
}

func (k KeyValueExpr) innerString(eol EOL, lvl int) string {
	return fmt.Sprintf("%s: %s", k.Key, innerString(eol, lvl, k.Value))
}

type ParenExpr struct {
	Expr Expr
}

func (p ParenExpr) expr() {}
func (p ParenExpr) String() string {
	return p.innerString(EOL(""), 0)
}

func (p ParenExpr) innerString(eol EOL, lvl int) string {
	return fmt.Sprintf("(%s)", innerString(eol, lvl, p.Expr))
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

func (b BinaryExpr) innerString(eol EOL, lvl int) string {
	return fmt.Sprintf("%s %s %s", innerString(eol, lvl, b.X), b.Op, innerString(eol, lvl+1, b.Y))
}

type TypeTransformExpr struct {
	Transform string // e.g. "Partial"
	Expr      Expr
}

func (tt TypeTransformExpr) expr() {}
func (tt TypeTransformExpr) String() string {
	return fmt.Sprintf("%s<%s>", tt.Transform, tt.Expr)
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
	Quot
	Value string
}

func (s Str) expr() {}
func (s Str) String() string {
	q := string(s.Quot)
	if q == "" {
		q = string(SingleQuot)

		if strings.Contains(s.Value, "\n") {
			q = string(BTickQuot)
		}
	}

	return q + s.Value + q
}

type VarDecl struct {
	Tok string

	Names
	Type  Expr
	Value Expr
}

func (v VarDecl) decl() {}
func (v VarDecl) String() string {
	tok := v.Tok
	if tok == "" {
		tok = "const"
	}

	return fmt.Sprintf("%s %s: %s = %s;", tok, v.Names, v.Type, v.Value)
}

type Type interface {
	Node
	typeName() string
}

var (
	_ Type = EnumType{}
	_ Type = InterfaceType{}
	_ Type = BasicType{}
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
	Extends []Expr
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
			b.WriteString(s.String())
		}
		b.WriteString(" ")
	}

	obj := ObjectLit{Elems: i.Elems, eol: EOLSemicolon}
	b.WriteString(obj.String())

	return b.String()
}

type ExportStmt struct {
	Decl Decl
}

func (e ExportStmt) decl() {}
func (e ExportStmt) String() string {
	return "export " + e.Decl.String()
}

// ListExpr represents lists in type definitions, like string[].
type ListExpr struct {
	Expr
}

func (l ListExpr) expr() {}
func (l ListExpr) String() string {
	return l.Expr.String() + "[]"
}
func (l ListExpr) innerString(eol EOL, lvl int) string {
	return innerString(eol, lvl, l.Expr) + "[]"
}

type ImportSpec struct {
	From Str
	Names
}

func (i ImportSpec) String() string {
	return fmt.Sprintf("import %s from %s;", i.Names, i.From)
}
