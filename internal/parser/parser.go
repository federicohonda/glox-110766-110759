package parser

import (
	"fmt"

	"github.com/federicohonda/glox-110766-110759/internal/ast"
	"github.com/federicohonda/glox-110766-110759/internal/token"
)

// ParseError es el error de sintaxis que se levanta con panic dentro del
// parser y se recupera en Parse. Guarda el token donde se detectó el
// problema para poder reportar la línea.
type ParseError struct {
	Token   token.Token
	Message string
}

func (e *ParseError) Error() string {
	if e.Token.Type == token.EOF {
		return fmt.Sprintf("[línea %d] Error de sintaxis al final: %s", e.Token.Line, e.Message)
	}
	return fmt.Sprintf("[línea %d] Error de sintaxis en '%s': %s", e.Token.Line, e.Token.Lexeme, e.Message)
}

// Parser recorre una lista de tokens ya escaneados y arma el AST de una
// expresión, siguiendo la gramática de precedencia:
//
//	expression → equality
//	equality   → comparison ( ( "!=" | "==" ) comparison )*
//	comparison → term ( ( ">" | ">=" | "<" | "<=" ) term )*
//	term       → factor ( ( "-" | "+" ) factor )*
//	factor     → unary ( ( "/" | "*" | "%" ) unary )*
//	unary      → ( "!" | "-" ) unary | primary
//	primary    → NUMBER | STRING | "true" | "false" | "nil" | "(" expression ")"
type Parser struct {
	tokens  []token.Token
	current int
}

func New(tokens []token.Token) *Parser {
	return &Parser{tokens: tokens}
}

// Parse parsea una única expresión y devuelve su AST. Ante un error de
// sintaxis, en vez de propagar (Expr, error) por cada función de la
// gramática, la función que lo detecta hace panic con un *ParseError y acá
// se lo recupera y convierte en el error devuelto — el mismo mecanismo que
// una excepción con catch en plox, adaptado a Go con panic/recover.
func (p *Parser) Parse() (expr ast.Expr, err error) {
	defer func() {
		if r := recover(); r != nil {
			parseErr, ok := r.(*ParseError)
			if !ok {
				panic(r)
			}
			err = parseErr
		}
	}()
	return p.expression(), nil
}

func (p *Parser) expression() ast.Expr {
	return p.equality()
}

func (p *Parser) equality() ast.Expr {
	expr := p.comparison()
	for p.match(token.BANG_EQUAL, token.EQUAL_EQUAL) {
		operator := p.previous()
		right := p.comparison()
		expr = &ast.Binary{Left: expr, Operator: operator, Right: right}
	}
	return expr
}

func (p *Parser) comparison() ast.Expr {
	expr := p.term()
	for p.match(token.GREATER, token.GREATER_EQUAL, token.LESS, token.LESS_EQUAL) {
		operator := p.previous()
		right := p.term()
		expr = &ast.Binary{Left: expr, Operator: operator, Right: right}
	}
	return expr
}

func (p *Parser) term() ast.Expr {
	expr := p.factor()
	for p.match(token.MINUS, token.PLUS) {
		operator := p.previous()
		right := p.factor()
		expr = &ast.Binary{Left: expr, Operator: operator, Right: right}
	}
	return expr
}

func (p *Parser) factor() ast.Expr {
	expr := p.unary()
	for p.match(token.SLASH, token.STAR, token.PERCENT) {
		operator := p.previous()
		right := p.unary()
		expr = &ast.Binary{Left: expr, Operator: operator, Right: right}
	}
	return expr
}

func (p *Parser) unary() ast.Expr {
	if p.match(token.BANG, token.MINUS) {
		operator := p.previous()
		right := p.unary()
		return &ast.Unary{Operator: operator, Right: right}
	}
	return p.primary()
}

func (p *Parser) primary() ast.Expr {
	switch {
	case p.match(token.FALSE):
		return &ast.Literal{Value: false}
	case p.match(token.TRUE):
		return &ast.Literal{Value: true}
	case p.match(token.NIL):
		return &ast.Literal{Value: nil}
	case p.match(token.NUMBER, token.STRING):
		return &ast.Literal{Value: p.previous().Literal}
	case p.match(token.LEFT_PAREN):
		expr := p.expression()
		p.consume(token.RIGHT_PAREN, "se esperaba ')' después de la expresión.")
		return &ast.Grouping{Expression: expr}
	}
	panic(p.errorAt(p.peek(), "se esperaba una expresión."))
}

func (p *Parser) match(types ...token.TokenType) bool {
	for _, t := range types {
		if p.check(t) {
			p.advance()
			return true
		}
	}
	return false
}

func (p *Parser) check(t token.TokenType) bool {
	if p.isAtEnd() {
		return false
	}
	return p.peek().Type == t
}

func (p *Parser) advance() token.Token {
	if !p.isAtEnd() {
		p.current++
	}
	return p.previous()
}

func (p *Parser) isAtEnd() bool {
	return p.peek().Type == token.EOF
}

func (p *Parser) peek() token.Token {
	return p.tokens[p.current]
}

func (p *Parser) previous() token.Token {
	return p.tokens[p.current-1]
}

func (p *Parser) consume(t token.TokenType, message string) token.Token {
	if p.check(t) {
		return p.advance()
	}
	panic(p.errorAt(p.peek(), message))
}

func (p *Parser) errorAt(tok token.Token, message string) *ParseError {
	return &ParseError{Token: tok, Message: message}
}
