package ast

import (
	"fmt"

	"github.com/federicohonda/glox-110766-110759/internal/token"
)

// Logical representa una expresión lógica `left (and | or) right`.
// Se modela en un nodo separado de Binary debido a que su semántica requiere
// evaluación por cortocircuito (short-circuiting).
type Logical struct {
	Left     Expr
	Operator token.Token
	Right    Expr
}

func (*Logical) isExpr() {}

func (l *Logical) String() string {
	return fmt.Sprintf("(%s %s %s)", l.Operator.Lexeme, l.Left, l.Right)
}
