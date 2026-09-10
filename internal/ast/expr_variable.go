package ast

import (
	"fmt"

	"github.com/federicohonda/glox-110766-110759/internal/token"
)

// Variable representa el acceso al valor de una variable en una expresión (ej. `x`).
type Variable struct {
	Name token.Token
}

func (*Variable) isExpr() {}

func (v *Variable) String() string {
	return v.Name.Lexeme
}

// Assign representa una asignación como expresión: `<name> = <value>`.
// En Lox, la asignación es una expresión que evalúa al valor asignado.
type Assign struct {
	Name  token.Token
	Value Expr
}

func (*Assign) isExpr() {}

func (a *Assign) String() string {
	return fmt.Sprintf("(= %s %s)", a.Name.Lexeme, a.Value)
}
