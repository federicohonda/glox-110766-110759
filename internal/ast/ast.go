package ast

import "github.com/federicohonda/glox-110766-110759/internal/token"

// Expr marca los nodos que representan una expresión de Lox. Es
// deliberadamente un marcador vacío: cómo se despacha por tipo de nodo
// (type switch vs visitor) es una decisión de la Fase 3, no de acá.
type Expr interface {
	isExpr()
}

// Binary representa `left operator right`, ej. `1 + 2`.
type Binary struct {
	Left     Expr
	Operator token.Token
	Right    Expr
}

// Grouping representa una expresión entre paréntesis, ej. `(1 + 2)`.
type Grouping struct {
	Expression Expr
}

// Literal representa un valor literal ya interpretado (no el lexeme crudo):
// float64, string, bool o nil.
type Literal struct {
	Value any
}

// Unary representa `operator right`, ej. `-1` o `!true`.
type Unary struct {
	Operator token.Token
	Right    Expr
}

func (*Binary) isExpr()   {}
func (*Grouping) isExpr() {}
func (*Literal) isExpr()  {}
func (*Unary) isExpr()    {}
