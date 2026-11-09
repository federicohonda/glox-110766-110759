package ast

import (
	"fmt"

	"github.com/federicohonda/glox-110766-110759/internal/token"
)

// Get representa el acceso a una propiedad: `<object>.<name>`.
type Get struct {
	Object Expr
	Name   token.Token
}

func (*Get) isExpr() {}

func (g *Get) String() string {
	return fmt.Sprintf("(. %s %s)", g.Object, g.Name.Lexeme)
}

// Set representa la asignación a una propiedad: `<object>.<name> = <value>`.
type Set struct {
	Object Expr
	Name   token.Token
	Value  Expr
}

func (*Set) isExpr() {}

func (s *Set) String() string {
	return fmt.Sprintf("(= (. %s %s) %s)", s.Object, s.Name.Lexeme, s.Value)
}

// This representa la referencia a la instancia receptora `this`.
type This struct {
	Keyword token.Token
}

func (*This) isExpr() {}

func (t *This) String() string {
	return "this"
}

// Super representa el acceso a un método de la superclase: `super.<method>`.
type Super struct {
	Keyword token.Token
	Method  token.Token
}

func (*Super) isExpr() {}

func (s *Super) String() string {
	return fmt.Sprintf("(super %s)", s.Method.Lexeme)
}
