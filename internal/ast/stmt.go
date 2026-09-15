package ast

import (
	"fmt"
	"strings"

	"github.com/federicohonda/glox-110766-110759/internal/token"
)

// Stmt marca los nodos que representan una sentencia (statement) en Lox.
// A diferencia de Expr (que produce un valor), una Stmt produce un efecto colateral.
type Stmt interface {
	isStmt()
	fmt.Stringer
}

// ExpressionStmt envuelve una expresión que se evalúa por sus efectos secundarios
// (por ejemplo, una asignación o llamada a función) descartando su valor.
type ExpressionStmt struct {
	Expression Expr
}

func (*ExpressionStmt) isStmt() {}

func (s *ExpressionStmt) String() string {
	return fmt.Sprintf("(expr %s)", s.Expression)
}

// PrintStmt representa la sentencia `print <expr>;`.
type PrintStmt struct {
	Expression Expr
}

func (*PrintStmt) isStmt() {}

func (s *PrintStmt) String() string {
	return fmt.Sprintf("(print %s)", s.Expression)
}

// VarDecl representa la declaración de una variable `var <name> = <initializer>;`.
// Si no tiene inicializador, Initializer es nil.
type VarDecl struct {
	Name        token.Token
	Initializer Expr
}

func (*VarDecl) isStmt() {}

func (s *VarDecl) String() string {
	if s.Initializer == nil {
		return fmt.Sprintf("(var %s)", s.Name.Lexeme)
	}
	return fmt.Sprintf("(var %s = %s)", s.Name.Lexeme, s.Initializer)
}

// Block representa un bloque de sentencias delimitado por llaves `{ <stmt>* }`.
type Block struct {
	Statements []Stmt
}

func (*Block) isStmt() {}

func (b *Block) String() string {
	var sb strings.Builder
	sb.WriteString("(block")
	for _, stmt := range b.Statements {
		sb.WriteString(" ")
		sb.WriteString(stmt.String())
	}
	sb.WriteString(")")
	return sb.String()
}

// IfStmt representa la sentencia condicional `if (<condition>) <thenBranch> else <elseBranch>`.
// ElseBranch es nil si no hay cláusula else.
type IfStmt struct {
	Condition  Expr
	ThenBranch Stmt
	ElseBranch Stmt
}

func (*IfStmt) isStmt() {}

func (s *IfStmt) String() string {
	if s.ElseBranch == nil {
		return fmt.Sprintf("(if %s %s)", s.Condition, s.ThenBranch)
	}
	return fmt.Sprintf("(if-else %s %s %s)", s.Condition, s.ThenBranch, s.ElseBranch)
}

// WhileStmt representa un bucle `while (<condition>) <body>`.
type WhileStmt struct {
	Condition Expr
	Body      Stmt
}

func (*WhileStmt) isStmt() {}

func (w *WhileStmt) String() string {
	return fmt.Sprintf("(while %s %s)", w.Condition, w.Body)
}

// FunDecl representa la declaración de una función
// `fun <name>(<params>) { <body> }`.
type FunDecl struct {
	Name   token.Token
	Params []token.Token
	Body   []Stmt
}

func (*FunDecl) isStmt() {}

func (f *FunDecl) String() string {
	params := make([]string, len(f.Params))
	for i, p := range f.Params {
		params[i] = p.Lexeme
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "(fun %s (%s)", f.Name.Lexeme, strings.Join(params, " "))
	for _, stmt := range f.Body {
		sb.WriteString(" ")
		sb.WriteString(stmt.String())
	}
	sb.WriteString(")")
	return sb.String()
}
