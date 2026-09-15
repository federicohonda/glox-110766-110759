package resolver

import (
	"fmt"

	"github.com/federicohonda/glox-110766-110759/internal/ast"
	"github.com/federicohonda/glox-110766-110759/internal/token"
)

// ResolveError es un error semántico detectado antes de ejecutar el programa.
type ResolveError struct {
	Token   token.Token
	Message string
}

func (e *ResolveError) Error() string {
	return fmt.Sprintf("[línea %d] Error semántico en '%s': %s", e.Token.Line, e.Token.Lexeme, e.Message)
}

type functionType int

const (
	noFunction functionType = iota
	insideFunction
)

// Resolver recorre el AST una única vez, antes de ejecutarlo, y calcula para
// cada uso de una variable local a cuántos entornos de distancia está su
// declaración. No evalúa ninguna expresión ni ejecuta ninguna sentencia: solo
// mira la estructura del programa, que es lo único que se conoce con certeza
// antes de correrlo.
type Resolver struct {
	// scopes es la pila de scopes locales abiertos. El valor indica si la
	// variable ya terminó de definirse (false = declarada, pero todavía se
	// está resolviendo su inicializador). El scope global no está en la pila:
	// una variable que no aparece en ningún scope se asume global.
	scopes          []map[string]bool
	locals          map[ast.Expr]int
	currentFunction functionType
	errors          []error
}

// Resolve devuelve la distancia de scope de cada expresión que usa una
// variable local (las globales no aparecen en el mapa) y la lista de errores
// semánticos encontrados. Las claves del mapa son punteros a los nodos, así
// que dos usos idénticos de la misma variable en lugares distintos del código
// son claves distintas.
func Resolve(stmts []ast.Stmt) (map[ast.Expr]int, []error) {
	r := &Resolver{locals: make(map[ast.Expr]int)}
	r.resolveStmts(stmts)
	return r.locals, r.errors
}

func (r *Resolver) resolveStmts(stmts []ast.Stmt) {
	for _, stmt := range stmts {
		r.resolveStmt(stmt)
	}
}

func (r *Resolver) resolveStmt(stmt ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.Block:
		r.beginScope()
		r.resolveStmts(s.Statements)
		r.endScope()

	case *ast.VarDecl:
		// Declarar antes de resolver el inicializador y definir después es lo
		// que permite detectar `var a = a;` dentro de un scope local.
		r.declare(s.Name)
		if s.Initializer != nil {
			r.resolveExpr(s.Initializer)
		}
		r.define(s.Name)

	case *ast.FunDecl:
		// El nombre se define antes de resolver el cuerpo para que la función
		// pueda llamarse a sí misma (recursión).
		r.declare(s.Name)
		r.define(s.Name)
		r.resolveFunction(s)

	case *ast.ExpressionStmt:
		r.resolveExpr(s.Expression)

	case *ast.PrintStmt:
		r.resolveExpr(s.Expression)

	case *ast.IfStmt:
		r.resolveExpr(s.Condition)
		r.resolveStmt(s.ThenBranch)
		if s.ElseBranch != nil {
			r.resolveStmt(s.ElseBranch)
		}

	case *ast.WhileStmt:
		r.resolveExpr(s.Condition)
		r.resolveStmt(s.Body)

	case *ast.ReturnStmt:
		if r.currentFunction == noFunction {
			r.report(s.Keyword, "no se puede usar 'return' fuera de una función.")
		}
		if s.Value != nil {
			r.resolveExpr(s.Value)
		}

	default:
		panic(fmt.Sprintf("resolver: tipo de sentencia no soportado: %T", stmt))
	}
}

func (r *Resolver) resolveExpr(expr ast.Expr) {
	switch e := expr.(type) {
	case *ast.Variable:
		if len(r.scopes) > 0 {
			if defined, declared := r.scopes[len(r.scopes)-1][e.Name.Lexeme]; declared && !defined {
				r.report(e.Name, "no se puede leer una variable local en su propio inicializador.")
			}
		}
		r.resolveLocal(e, e.Name)

	case *ast.Assign:
		r.resolveExpr(e.Value)
		r.resolveLocal(e, e.Name)

	case *ast.Binary:
		r.resolveExpr(e.Left)
		r.resolveExpr(e.Right)

	case *ast.Logical:
		r.resolveExpr(e.Left)
		r.resolveExpr(e.Right)

	case *ast.Unary:
		r.resolveExpr(e.Right)

	case *ast.Grouping:
		r.resolveExpr(e.Expression)

	case *ast.Call:
		r.resolveExpr(e.Callee)
		for _, arg := range e.Arguments {
			r.resolveExpr(arg)
		}

	case *ast.Literal:

	default:
		panic(fmt.Sprintf("resolver: tipo de expresión no soportado: %T", expr))
	}
}

// resolveFunction abre un único scope para parámetros y cuerpo, igual que
// Function.Call en el intérprete, que ejecuta el cuerpo en el mismo entorno
// donde define los parámetros. Si no coincidieran, las distancias calculadas
// quedarían corridas en uno.
func (r *Resolver) resolveFunction(fn *ast.FunDecl) {
	enclosing := r.currentFunction
	r.currentFunction = insideFunction

	r.beginScope()
	for _, param := range fn.Params {
		r.declare(param)
		r.define(param)
	}
	r.resolveStmts(fn.Body)
	r.endScope()

	r.currentFunction = enclosing
}

// resolveLocal busca la variable desde el scope más interno hacia afuera y
// registra a cuántos scopes de distancia la encontró. Si no está en ninguno,
// no registra nada: el intérprete la va a buscar en los globales.
func (r *Resolver) resolveLocal(expr ast.Expr, name token.Token) {
	for i := len(r.scopes) - 1; i >= 0; i-- {
		if _, ok := r.scopes[i][name.Lexeme]; ok {
			r.locals[expr] = len(r.scopes) - 1 - i
			return
		}
	}
}

func (r *Resolver) beginScope() {
	r.scopes = append(r.scopes, make(map[string]bool))
}

func (r *Resolver) endScope() {
	r.scopes = r.scopes[:len(r.scopes)-1]
}

// declare no hace nada en el scope global: ahí Lox permite redeclarar
// variables (útil en el REPL). En un scope local, redeclarar es un error.
func (r *Resolver) declare(name token.Token) {
	if len(r.scopes) == 0 {
		return
	}
	scope := r.scopes[len(r.scopes)-1]
	if _, exists := scope[name.Lexeme]; exists {
		r.report(name, "ya existe una variable con este nombre en este scope.")
	}
	scope[name.Lexeme] = false
}

func (r *Resolver) define(name token.Token) {
	if len(r.scopes) == 0 {
		return
	}
	r.scopes[len(r.scopes)-1][name.Lexeme] = true
}

func (r *Resolver) report(tok token.Token, message string) {
	r.errors = append(r.errors, &ResolveError{Token: tok, Message: message})
}
