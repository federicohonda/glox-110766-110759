package scanner

import (
	"testing"

	"github.com/federicohonda/glox-110766-110759/internal/token"
)

func TestSingleCharacterTokens(t *testing.T) {
	input := "(){},-+;*%"
	sc := New(input)
	tokens, errs := sc.Scan()

	if len(errs) > 0 {
		t.Fatalf("se encontraron errores inesperados: %v", errs)
	}

	expectedTypes := []token.TokenType{
		token.LEFT_PAREN,
		token.RIGHT_PAREN,
		token.LEFT_BRACE,
		token.RIGHT_BRACE,
		token.COMMA,
		token.MINUS,
		token.PLUS,
		token.SEMICOLON,
		token.STAR,
		token.PERCENT,
		token.EOF,
	}

	if len(tokens) != len(expectedTypes) {
		t.Fatalf("se esperaban %d tokens, se obtuvieron %d", len(expectedTypes), len(tokens))
	}

	for i, expected := range expectedTypes {
		if tokens[i].Type != expected {
			t.Errorf("token %d: se esperaba tipo %s, se obtuvo %s", i, expected, tokens[i].Type)
		}
	}
}

func TestOperatorsAndComments(t *testing.T) {
	input := "! != = == < <= > >= / // esto es un comentario\n/"
	sc := New(input)
	tokens, errs := sc.Scan()

	if len(errs) > 0 {
		t.Fatalf("se encontraron errores inesperados: %v", errs)
	}

	expectedTypes := []token.TokenType{
		token.BANG,
		token.BANG_EQUAL,
		token.EQUAL,
		token.EQUAL_EQUAL,
		token.LESS,
		token.LESS_EQUAL,
		token.GREATER,
		token.GREATER_EQUAL,
		token.SLASH,
		token.SLASH,
		token.EOF,
	}

	if len(tokens) != len(expectedTypes) {
		t.Fatalf("se esperaban %d tokens, se obtuvieron %d", len(expectedTypes), len(tokens))
	}

	for i, expected := range expectedTypes {
		if tokens[i].Type != expected {
			t.Errorf("token %d: se esperaba tipo %s, se obtuvo %s", i, expected, tokens[i].Type)
		}
	}

	// El segundo SLASH debe estar en la línea 2
	if tokens[9].Line != 2 {
		t.Errorf("se esperaba que el token SLASH esté en la línea 2, pero está en la línea %d", tokens[9].Line)
	}
}

func TestUnexpectedCharacter(t *testing.T) {
	input := "@"
	sc := New(input)
	_, errs := sc.Scan()

	if len(errs) != 1 {
		t.Fatalf("se esperaba 1 error, se obtuvieron %d: %v", len(errs), errs)
	}
}
