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

func TestLiterals(t *testing.T) {
	input := `123 45.67 "hola mundo"`
	sc := New(input)
	tokens, errs := sc.Scan()

	if len(errs) > 0 {
		t.Fatalf("se encontraron errores inesperados: %v", errs)
	}

	if len(tokens) != 4 { // 3 literales + EOF
		t.Fatalf("se esperaban 4 tokens, se obtuvieron %d", len(tokens))
	}

	// 123
	if tokens[0].Type != token.NUMBER || tokens[0].Literal != 123.0 {
		t.Errorf("token 0: se esperaba NUMBER(123.0), se obtuvo %v", tokens[0])
	}

	// 45.67
	if tokens[1].Type != token.NUMBER || tokens[1].Literal != 45.67 {
		t.Errorf("token 1: se esperaba NUMBER(45.67), se obtuvo %v", tokens[1])
	}

	// "hola mundo"
	if tokens[2].Type != token.STRING || tokens[2].Literal != "hola mundo" {
		t.Errorf("token 2: se esperaba STRING(\"hola mundo\"), se obtuvo %v", tokens[2])
	}
}

func TestMultilineString(t *testing.T) {
	input := "\"linea 1\nlinea 2\nlinea 3\""
	sc := New(input)
	tokens, errs := sc.Scan()

	if len(errs) > 0 {
		t.Fatalf("se encontraron errores inesperados: %v", errs)
	}

	if len(tokens) != 2 {
		t.Fatalf("se esperaban 2 tokens, se obtuvieron %d", len(tokens))
	}

	if tokens[0].Type != token.STRING {
		t.Errorf("se esperaba tipo STRING, se obtuvo %s", tokens[0].Type)
	}

	expectedVal := "linea 1\nlinea 2\nlinea 3"
	if tokens[0].Literal != expectedVal {
		t.Errorf("se esperaba contenido %q, se obtuvo %q", expectedVal, tokens[0].Literal)
	}

	if tokens[0].Line != 3 {
		t.Errorf("se esperaba que la línea final sea 3, se obtuvo %d", tokens[0].Line)
	}
}
