package bytecode

import (
	"github.com/federicohonda/glox-110766-110759/internal/ast"
)

// findCaptured recorre el programa antes de compilarlo y devuelve las
// declaraciones locales que se usan desde una función anidada. Las claves son
// los nodos de la declaración: *ast.VarDecl, *ast.FunDecl, o *token.Token
// (apuntando adentro de FunDecl.Params) para un parámetro.
//
// Hace falta saberlo antes de emitir porque esas locales se guardan en una
// celda desde su declaración (OpBox), y la declaración se compila antes que
// la función que la captura. Los scopes se abren y se declaran en el mismo
// orden que en el compilador, así que un nombre se resuelve a la misma
// declaración en los dos (incluido el caso del bug de closures: un uso que
// aparece antes de la declaración local queda como global).
func findCaptured(stmts []ast.Stmt) map[any]bool {
	f := &captureFinder{captured: make(map[any]bool)}
	f.stmts(stmts)
	return f.captured
}

type captureScope struct {
	// function es la profundidad de función en la que se abrió el scope.
	function int
	names    map[string]any
}

type captureFinder struct {
	// scopes son los scopes locales abiertos; el global no está, igual que en
	// el resolver: un nombre que no aparece en ninguno es global.
	scopes   []captureScope
	function int
	captured map[any]bool
}

func (f *captureFinder) beginScope() {
	f.scopes = append(f.scopes, captureScope{function: f.function, names: make(map[string]any)})
}

func (f *captureFinder) endScope() {
	f.scopes = f.scopes[:len(f.scopes)-1]
}

func (f *captureFinder) declare(name string, decl any) {
	if len(f.scopes) > 0 {
		f.scopes[len(f.scopes)-1].names[name] = decl
	}
}

// use marca la declaración a la que se resuelve name si es de una función
// más externa que la actual.
func (f *captureFinder) use(name string) {
	for i := len(f.scopes) - 1; i >= 0; i-- {
		if decl, ok := f.scopes[i].names[name]; ok {
			if f.scopes[i].function < f.function {
				f.captured[decl] = true
			}
			return
		}
	}
}

func (f *captureFinder) stmts(stmts []ast.Stmt) {
	for _, stmt := range stmts {
		f.stmt(stmt)
	}
}

func (f *captureFinder) stmt(stmt ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.PrintStmt:
		f.expr(s.Expression)
	case *ast.ExpressionStmt:
		f.expr(s.Expression)
	case *ast.VarDecl:
		if s.Initializer != nil {
			f.expr(s.Initializer)
		}
		f.declare(s.Name.Lexeme, s)
	case *ast.Block:
		f.beginScope()
		f.stmts(s.Statements)
		f.endScope()
	case *ast.IfStmt:
		f.expr(s.Condition)
		f.stmt(s.ThenBranch)
		if s.ElseBranch != nil {
			f.stmt(s.ElseBranch)
		}
	case *ast.WhileStmt:
		f.expr(s.Condition)
		f.stmt(s.Body)
	case *ast.ReturnStmt:
		if s.Value != nil {
			f.expr(s.Value)
		}
	case *ast.FunDecl:
		// El nombre se declara antes del cuerpo: una función local recursiva
		// se captura a sí misma.
		f.declare(s.Name.Lexeme, s)
		f.function++
		f.beginScope()
		for i := range s.Params {
			f.declare(s.Params[i].Lexeme, &s.Params[i])
		}
		f.stmts(s.Body)
		f.endScope()
		f.function--
	}
}

func (f *captureFinder) expr(expr ast.Expr) {
	switch e := expr.(type) {
	case *ast.Grouping:
		f.expr(e.Expression)
	case *ast.Unary:
		f.expr(e.Right)
	case *ast.Binary:
		f.expr(e.Left)
		f.expr(e.Right)
	case *ast.Logical:
		f.expr(e.Left)
		f.expr(e.Right)
	case *ast.Variable:
		f.use(e.Name.Lexeme)
	case *ast.Assign:
		f.expr(e.Value)
		f.use(e.Name.Lexeme)
	case *ast.Call:
		f.expr(e.Callee)
		for _, arg := range e.Arguments {
			f.expr(arg)
		}
	}
}
