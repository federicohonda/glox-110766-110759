package parser

import (
	"fmt"

	"github.com/federicohonda/glox-110766-110759/internal/ast"
	"github.com/federicohonda/glox-110766-110759/internal/token"
)

// ParseError es el error de sintaxis que se levanta con panic dentro del
// parser y se recupera en Parse o declaration. Guarda el token donde se
// detectó el problema para reportar la línea y contexto.
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

// Parser recorre una lista de tokens ya escaneados y arma el AST de un
// programa de Lox, siguiendo la gramática de precedencia y sentencias:
//
//	program        → declaration* EOF
//	declaration    → varDecl | statement
//	varDecl        → "var" IDENTIFIER ( "=" expression )? ";"
//	statement      → printStmt | block | exprStmt
//	printStmt      → "print" expression ";"
//	exprStmt       → expression ";"
//	block          → "{" declaration* "}"
//	expression     → assignment
//	assignment     → IDENTIFIER "=" assignment | equality
//	equality       → comparison ( ( "!=" | "==" ) comparison )*
//	comparison     → term ( ( ">" | ">=" | "<" | "<=" ) term )*
//	term           → factor ( ( "-" | "+" ) factor )*
//	factor         → unary ( ( "/" | "*" | "%" ) unary )*
//	unary          → ( "!" | "-" ) unary | primary
//	primary        → NUMBER | STRING | "true" | "false" | "nil" | "(" expression ")" | IDENTIFIER
type Parser struct {
	tokens  []token.Token
	current int
}

func New(tokens []token.Token) *Parser {
	return &Parser{tokens: tokens}
}

// Parse parsea una secuencia completa de declaraciones/sentencias hasta el EOF.
// Si ocurren errores de sintaxis, se sincroniza para acumular todos los errores
// posibles y devolver la lista de errores encontrados.
func (p *Parser) Parse() ([]ast.Stmt, []error) {
	var statements []ast.Stmt
	var errors []error

	for !p.isAtEnd() {
		stmt, err := p.declaration()
		if err != nil {
			errors = append(errors, err)
			p.synchronize()
		} else {
			statements = append(statements, stmt)
		}
	}

	if len(errors) > 0 {
		return nil, errors
	}
	return statements, nil
}

// ParseExpression parsea una única expresión aislada (útil para tests unitarios
// o evaluación de expresiones sueltas).
func (p *Parser) ParseExpression() (expr ast.Expr, err error) {
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

func (p *Parser) declaration() (stmt ast.Stmt, err error) {
	defer func() {
		if r := recover(); r != nil {
			if parseErr, ok := r.(*ParseError); ok {
				err = parseErr
			} else {
				panic(r)
			}
		}
	}()

	if p.match(token.VAR) {
		return p.varDeclaration(), nil
	}
	return p.statement(), nil
}

func (p *Parser) varDeclaration() ast.Stmt {
	name := p.consume(token.IDENTIFIER, "se esperaba el nombre de la variable.")

	var initializer ast.Expr
	if p.match(token.EQUAL) {
		initializer = p.expression()
	}

	p.consume(token.SEMICOLON, "se esperaba ';' después de la declaración de la variable.")
	return &ast.VarDecl{Name: name, Initializer: initializer}
}

func (p *Parser) statement() ast.Stmt {
	if p.match(token.IF) {
		return p.ifStatement()
	}
	if p.match(token.PRINT) {
		return p.printStatement()
	}
	if p.match(token.LEFT_BRACE) {
		return &ast.Block{Statements: p.block()}
	}
	return p.expressionStatement()
}

func (p *Parser) ifStatement() ast.Stmt {
	p.consume(token.LEFT_PAREN, "se esperaba '(' después de 'if'.")
	condition := p.expression()
	p.consume(token.RIGHT_PAREN, "se esperaba ')' después de la condición del if.")

	thenBranch := p.statement()
	var elseBranch ast.Stmt
	if p.match(token.ELSE) {
		elseBranch = p.statement()
	}

	return &ast.IfStmt{
		Condition:  condition,
		ThenBranch: thenBranch,
		ElseBranch: elseBranch,
	}
}

func (p *Parser) printStatement() ast.Stmt {
	value := p.expression()
	p.consume(token.SEMICOLON, "se esperaba ';' después del valor a imprimir.")
	return &ast.PrintStmt{Expression: value}
}

func (p *Parser) expressionStatement() ast.Stmt {
	expr := p.expression()
	p.consume(token.SEMICOLON, "se esperaba ';' después de la expresión.")
	return &ast.ExpressionStmt{Expression: expr}
}

func (p *Parser) block() []ast.Stmt {
	var statements []ast.Stmt

	for !p.check(token.RIGHT_BRACE) && !p.isAtEnd() {
		stmt, err := p.declaration()
		if err != nil {
			panic(err)
		}
		statements = append(statements, stmt)
	}

	p.consume(token.RIGHT_BRACE, "se esperaba '}' después del bloque.")
	return statements
}

func (p *Parser) expression() ast.Expr {
	return p.assignment()
}

func (p *Parser) assignment() ast.Expr {
	expr := p.logicOr()

	if p.match(token.EQUAL) {
		equals := p.previous()
		value := p.assignment()

		if variable, ok := expr.(*ast.Variable); ok {
			return &ast.Assign{Name: variable.Name, Value: value}
		}

		panic(p.errorAt(equals, "destino de asignación inválido."))
	}

	return expr
}

func (p *Parser) logicOr() ast.Expr {
	expr := p.logicAnd()

	for p.match(token.OR) {
		operator := p.previous()
		right := p.logicAnd()
		expr = &ast.Logical{Left: expr, Operator: operator, Right: right}
	}

	return expr
}

func (p *Parser) logicAnd() ast.Expr {
	expr := p.equality()

	for p.match(token.AND) {
		operator := p.previous()
		right := p.equality()
		expr = &ast.Logical{Left: expr, Operator: operator, Right: right}
	}

	return expr
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
	case p.match(token.IDENTIFIER):
		return &ast.Variable{Name: p.previous()}
	case p.match(token.LEFT_PAREN):
		expr := p.expression()
		p.consume(token.RIGHT_PAREN, "se esperaba ')' después de la expresión.")
		return &ast.Grouping{Expression: expr}
	}
	panic(p.errorAt(p.peek(), "se esperaba una expresión."))
}

func (p *Parser) synchronize() {
	p.advance()

	for !p.isAtEnd() {
		if p.previous().Type == token.SEMICOLON {
			return
		}

		switch p.peek().Type {
		case token.VAR, token.PRINT, token.IF, token.WHILE, token.FOR, token.FUN, token.RETURN:
			return
		}

		p.advance()
	}
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
