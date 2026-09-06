package ast_test

import (
	"testing"

	"github.com/federicohonda/glox-110766-110759/internal/ast"
	"github.com/federicohonda/glox-110766-110759/internal/token"
)

// TestBuildTreeByHand arma a mano el AST de `1 + 2` y verifica que las
// structs se puedan anidar sin explotar, tal como pide el checkpoint de la
// Fase 2 - Parte 1 del plan.
func TestBuildTreeByHand(t *testing.T) {
	expr := &ast.Binary{
		Left:     &ast.Literal{Value: 1.0},
		Operator: token.Token{Type: token.PLUS, Lexeme: "+", Line: 1},
		Right:    &ast.Literal{Value: 2.0},
	}

	left, ok := expr.Left.(*ast.Literal)
	if !ok || left.Value != 1.0 {
		t.Fatalf("Left esperado *ast.Literal{1.0}, obtuve %#v", expr.Left)
	}

	right, ok := expr.Right.(*ast.Literal)
	if !ok || right.Value != 2.0 {
		t.Fatalf("Right esperado *ast.Literal{2.0}, obtuve %#v", expr.Right)
	}

	if expr.Operator.Type != token.PLUS {
		t.Fatalf("Operator esperado PLUS, obtuve %v", expr.Operator.Type)
	}
}

// TestNestedExpr verifica que se pueda anidar Grouping y Unary sobre un
// Binary, ej. `-(1 + 2)`.
func TestNestedExpr(t *testing.T) {
	inner := &ast.Binary{
		Left:     &ast.Literal{Value: 1.0},
		Operator: token.Token{Type: token.PLUS, Lexeme: "+", Line: 1},
		Right:    &ast.Literal{Value: 2.0},
	}

	var expr ast.Expr = &ast.Unary{
		Operator: token.Token{Type: token.MINUS, Lexeme: "-", Line: 1},
		Right:    &ast.Grouping{Expression: inner},
	}

	unary, ok := expr.(*ast.Unary)
	if !ok {
		t.Fatalf("esperaba *ast.Unary, obtuve %#v", expr)
	}

	grouping, ok := unary.Right.(*ast.Grouping)
	if !ok || grouping.Expression != ast.Expr(inner) {
		t.Fatalf("esperaba Grouping envolviendo el Binary interno, obtuve %#v", unary.Right)
	}
}
