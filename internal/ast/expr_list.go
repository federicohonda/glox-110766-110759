package ast

import (
	"fmt"
	"strings"

	"github.com/federicohonda/glox-110766-110759/internal/token"
)

// ListLiteral representa una lista escrita en el código: `[a, b, c]`.
// Bracket es el `[` de apertura, para reportar la línea.
type ListLiteral struct {
	Bracket  token.Token
	Elements []Expr
}

func (*ListLiteral) isExpr() {}

func (l *ListLiteral) String() string {
	var sb strings.Builder
	sb.WriteString("(list")
	for _, e := range l.Elements {
		sb.WriteString(" ")
		sb.WriteString(e.String())
	}
	sb.WriteString(")")
	return sb.String()
}

// Index representa la lectura de un elemento: `<object>[<index>]`. Bracket es
// el `]` de cierre, igual que Call guarda el `)`.
type Index struct {
	Object  Expr
	Bracket token.Token
	Index   Expr
}

func (*Index) isExpr() {}

func (i *Index) String() string {
	return fmt.Sprintf("([] %s %s)", i.Object, i.Index)
}

// SetIndex representa la asignación a un elemento: `<object>[<index>] = <value>`.
type SetIndex struct {
	Object  Expr
	Bracket token.Token
	Index   Expr
	Value   Expr
}

func (*SetIndex) isExpr() {}

func (s *SetIndex) String() string {
	return fmt.Sprintf("(= ([] %s %s) %s)", s.Object, s.Index, s.Value)
}
