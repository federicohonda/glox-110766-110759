package ast

import (
	"fmt"

	"github.com/federicohonda/glox-110766-110759/internal/token"
)

// Expr marca los nodos que representan una expresión de Lox. Más allá de
// poder imprimirse (fmt.Stringer, para el modo --parsing), es deliberadamente
// un marcador vacío: cómo se despacha por tipo de nodo al evaluar (type
// switch vs visitor) es una decisión de la Fase 3, no de acá.
type Expr interface {
	isExpr()
	fmt.Stringer
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

// String imprime el árbol en notación prefija parentizada (ej. `(+ 1 (* 2 3))`
// para `1 + 2 * 3`), de forma que el anidamiento y la precedencia se lean
// directamente en la salida — es el formato que usa el modo `--parsing`.
func (b *Binary) String() string {
	return fmt.Sprintf("(%s %s %s)", b.Operator.Lexeme, b.Left, b.Right)
}

func (g *Grouping) String() string {
	return fmt.Sprintf("(group %s)", g.Expression)
}

func (l *Literal) String() string {
	if l.Value == nil {
		return "nil"
	}
	return fmt.Sprintf("%v", l.Value)
}

func (u *Unary) String() string {
	return fmt.Sprintf("(%s %s)", u.Operator.Lexeme, u.Right)
}
