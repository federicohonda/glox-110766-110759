package ast

import (
	"fmt"
	"strings"

	"github.com/federicohonda/glox-110766-110759/internal/token"
)

// Call representa una invocación `callee(arg1, arg2, ...)`. Callee es una
// expresión cualquiera (no solo un identificador), lo que permite encadenar
// llamadas como `f()()`. Paren es el ')' de cierre, que se guarda para
// reportar la línea de los errores de runtime de la llamada.
type Call struct {
	Callee    Expr
	Paren     token.Token
	Arguments []Expr
}

func (*Call) isExpr() {}

func (c *Call) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "(call %s", c.Callee)
	for _, arg := range c.Arguments {
		sb.WriteString(" ")
		sb.WriteString(arg.String())
	}
	sb.WriteString(")")
	return sb.String()
}
