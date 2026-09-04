package scanner

import (
	"fmt"

	"github.com/federicohonda/glox-110766-110759/internal/token"
)

// Scanner se encarga del análisis léxico convirtiendo el código fuente en una secuencia de tokens.
type Scanner struct {
	source  string
	tokens  []token.Token
	start   int
	current int
	line    int
	errors  []string
}

// New crea e inicializa un nuevo Scanner para el código fuente provisto.
func New(source string) *Scanner {
	return &Scanner{
		source: source,
		line:   1,
	}
}

// Scan analiza la totalidad del código fuente y devuelve los tokens reconocidos y los errores acumulados.
func (s *Scanner) Scan() ([]token.Token, []string) {
	s.start = 0
	s.current = 0
	s.line = 1
	s.tokens = nil
	s.errors = nil

	for !s.isAtEnd() {
		s.start = s.current
		s.scanToken()
	}

	s.tokens = append(s.tokens, token.Token{
		Type:    token.EOF,
		Lexeme:  "",
		Literal: nil,
		Line:    s.line,
	})

	return s.tokens, s.errors
}

// HasErrors indica si se encontraron errores durante el escaneo.
func (s *Scanner) HasErrors() bool {
	return len(s.errors) > 0
}

// Errors devuelve la lista de errores encontrados.
func (s *Scanner) Errors() []string {
	return s.errors
}

// Tokens devuelve la lista de tokens generados.
func (s *Scanner) Tokens() []token.Token {
	return s.tokens
}

func (s *Scanner) scanToken() {
	c := s.advance()

	switch c {
	// Tokens de un solo carácter
	case '(':
		s.addToken(token.LEFT_PAREN)
	case ')':
		s.addToken(token.RIGHT_PAREN)
	case '{':
		s.addToken(token.LEFT_BRACE)
	case '}':
		s.addToken(token.RIGHT_BRACE)
	case ',':
		s.addToken(token.COMMA)
	case '-':
		s.addToken(token.MINUS)
	case '+':
		s.addToken(token.PLUS)
	case ';':
		s.addToken(token.SEMICOLON)
	case '*':
		s.addToken(token.STAR)
	case '%':
		s.addToken(token.PERCENT)

	// Operadores de uno o dos caracteres
	case '!':
		if s.match('=') {
			s.addToken(token.BANG_EQUAL)
		} else {
			s.addToken(token.BANG)
		}
	case '=':
		if s.match('=') {
			s.addToken(token.EQUAL_EQUAL)
		} else {
			s.addToken(token.EQUAL)
		}
	case '<':
		if s.match('=') {
			s.addToken(token.LESS_EQUAL)
		} else {
			s.addToken(token.LESS)
		}
	case '>':
		if s.match('=') {
			s.addToken(token.GREATER_EQUAL)
		} else {
			s.addToken(token.GREATER)
		}

	// Barras y comentarios
	case '/':
		if s.match('/') {
			// Comentario de una línea: consumir hasta el final de la línea o del archivo.
			// No consumimos el '\n' para que el loop principal actualice s.line.
			for s.peek() != '\n' && !s.isAtEnd() {
				s.advance()
			}
		} else {
			s.addToken(token.SLASH)
		}

	// Espacios en blanco y saltos de línea
	case ' ', '\r', '\t':
		// Ignorar whitespace
	case '\n':
		s.line++

	default:
		s.addError(fmt.Sprintf("[línea %d] Error: Carácter no reconocido: %q.", s.line, c))
	}
}

func (s *Scanner) isAtEnd() bool {
	return s.current >= len(s.source)
}

func (s *Scanner) advance() byte {
	c := s.source[s.current]
	s.current++
	return c
}

func (s *Scanner) match(expected byte) bool {
	if s.isAtEnd() {
		return false
	}
	if s.source[s.current] != expected {
		return false
	}
	s.current++
	return true
}

func (s *Scanner) peek() byte {
	if s.isAtEnd() {
		return 0
	}
	return s.source[s.current]
}

func (s *Scanner) addToken(tokenType token.TokenType) {
	s.addTokenLiteral(tokenType, nil)
}

func (s *Scanner) addTokenLiteral(tokenType token.TokenType, literal any) {
	text := s.source[s.start:s.current]
	s.tokens = append(s.tokens, token.Token{
		Type:    tokenType,
		Lexeme:  text,
		Literal: literal,
		Line:    s.line,
	})
}

func (s *Scanner) addError(msg string) {
	s.errors = append(s.errors, msg)
}
