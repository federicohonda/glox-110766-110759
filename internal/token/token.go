package token

import "fmt"

type TokenType int

const (
	LEFT_PAREN TokenType = iota
	RIGHT_PAREN
	LEFT_BRACE
	RIGHT_BRACE
	COMMA
	MINUS
	PLUS
	SEMICOLON
	STAR
	PERCENT
	SLASH
	BANG
	BANG_EQUAL
	EQUAL
	EQUAL_EQUAL
	GREATER
	GREATER_EQUAL
	LESS
	LESS_EQUAL
	IDENTIFIER
	STRING
	NUMBER
	AND
	ELSE
	FALSE
	FUN
	FOR
	IF
	NIL
	OR
	PRINT
	RETURN
	TRUE
	VAR
	WHILE
	EOF
)

// tokenTypeNames debe mantenerse en el mismo orden que la lista de constantes de arriba.
var tokenTypeNames = [...]string{
	"LEFT_PAREN",
	"RIGHT_PAREN",
	"LEFT_BRACE",
	"RIGHT_BRACE",
	"COMMA",
	"MINUS",
	"PLUS",
	"SEMICOLON",
	"STAR",
	"PERCENT",
	"SLASH",
	"BANG",
	"BANG_EQUAL",
	"EQUAL",
	"EQUAL_EQUAL",
	"GREATER",
	"GREATER_EQUAL",
	"LESS",
	"LESS_EQUAL",
	"IDENTIFIER",
	"STRING",
	"NUMBER",
	"AND",
	"ELSE",
	"FALSE",
	"FUN",
	"FOR",
	"IF",
	"NIL",
	"OR",
	"PRINT",
	"RETURN",
	"TRUE",
	"VAR",
	"WHILE",
	"EOF",
}

func (t TokenType) String() string {
	if t < 0 || int(t) >= len(tokenTypeNames) {
		return fmt.Sprintf("TokenType(%d)", int(t))
	}
	return tokenTypeNames[t]
}

type Token struct {
	Type    TokenType
	Lexeme  string
	Literal any // float64 | string | bool | nil
	Line    int
}

func (t Token) String() string {
	return fmt.Sprintf("%s %q %v", t.Type, t.Lexeme, t.Literal)
}
