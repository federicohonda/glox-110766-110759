package resolver_test

import (
	"strings"
	"testing"

	"github.com/federicohonda/glox-110766-110759/internal/ast"
	"github.com/federicohonda/glox-110766-110759/internal/parser"
	"github.com/federicohonda/glox-110766-110759/internal/resolver"
	"github.com/federicohonda/glox-110766-110759/internal/scanner"
)

func parse(t *testing.T, source string) []ast.Stmt {
	t.Helper()
	tokens, scanErrs := scanner.New(source).Scan()
	if len(scanErrs) > 0 {
		t.Fatalf("error de escaneo inesperado en %q: %v", source, scanErrs)
	}
	stmts, parseErrs := parser.New(tokens).Parse()
	if len(parseErrs) > 0 {
		t.Fatalf("error de parseo inesperado en %q: %v", source, parseErrs)
	}
	return stmts
}

// distances resuelve el programa y devuelve, por nombre de variable, la lista
// de distancias en el orden en que aparecen los usos resueltos como locales.
// Los usos globales se marcan con -1.
func distances(t *testing.T, source string) map[string][]int {
	t.Helper()
	stmts := parse(t, source)
	locals, errs := resolver.Resolve(stmts)
	if len(errs) > 0 {
		t.Fatalf("errores semánticos inesperados en %q: %v", source, errs)
	}

	got := make(map[string][]int)
	var visitExpr func(ast.Expr)
	var visitStmt func(ast.Stmt)
	record := func(expr ast.Expr, name string) {
		if d, ok := locals[expr]; ok {
			got[name] = append(got[name], d)
		} else {
			got[name] = append(got[name], -1)
		}
	}
	visitExpr = func(expr ast.Expr) {
		switch e := expr.(type) {
		case *ast.Variable:
			record(e, e.Name.Lexeme)
		case *ast.Assign:
			visitExpr(e.Value)
			record(e, e.Name.Lexeme)
		case *ast.Binary:
			visitExpr(e.Left)
			visitExpr(e.Right)
		case *ast.Logical:
			visitExpr(e.Left)
			visitExpr(e.Right)
		case *ast.Unary:
			visitExpr(e.Right)
		case *ast.Grouping:
			visitExpr(e.Expression)
		case *ast.Call:
			visitExpr(e.Callee)
			for _, a := range e.Arguments {
				visitExpr(a)
			}
		}
	}
	visitStmt = func(stmt ast.Stmt) {
		switch s := stmt.(type) {
		case *ast.Block:
			for _, inner := range s.Statements {
				visitStmt(inner)
			}
		case *ast.VarDecl:
			if s.Initializer != nil {
				visitExpr(s.Initializer)
			}
		case *ast.FunDecl:
			for _, inner := range s.Body {
				visitStmt(inner)
			}
		case *ast.ExpressionStmt:
			visitExpr(s.Expression)
		case *ast.PrintStmt:
			visitExpr(s.Expression)
		case *ast.IfStmt:
			visitExpr(s.Condition)
			visitStmt(s.ThenBranch)
			if s.ElseBranch != nil {
				visitStmt(s.ElseBranch)
			}
		case *ast.WhileStmt:
			visitExpr(s.Condition)
			visitStmt(s.Body)
		case *ast.ReturnStmt:
			if s.Value != nil {
				visitExpr(s.Value)
			}
		}
	}
	for _, s := range stmts {
		visitStmt(s)
	}
	return got
}

func TestResolveDistances(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   map[string][]int
	}{
		{
			name:   "una variable global no se registra",
			source: "var a = 1; print a; a = 2;",
			want:   map[string][]int{"a": {-1, -1}},
		},
		{
			name:   "local en el mismo bloque está a distancia 0",
			source: "{ var a = 1; print a; }",
			want:   map[string][]int{"a": {0}},
		},
		{
			name:   "cada bloque de anidamiento suma uno",
			source: "{ var a = 1; { { print a; } } }",
			want:   map[string][]int{"a": {2}},
		},
		{
			name:   "el shadowing apunta a la declaración más cercana",
			source: "{ var a = 1; { var a = 2; print a; } print a; }",
			want:   map[string][]int{"a": {0, 0}},
		},
		{
			name:   "parámetros y cuerpo comparten un único scope",
			source: "fun f(x) { var y = x; print y; }",
			want:   map[string][]int{"x": {0}, "y": {0}},
		},
		{
			name:   "variable capturada por una función interna",
			source: "fun externa() { var i = 0; fun interna() { i = i + 1; } }",
			want:   map[string][]int{"i": {1, 1}},
		},
		{
			name:   "la recursión de una función global usa el global",
			source: "fun fib(n) { if (n <= 1) return n; return fib(n - 1); }",
			want:   map[string][]int{"n": {0, 0, 0}, "fib": {-1}},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := distances(t, c.source)
			for name, want := range c.want {
				if !equalInts(got[name], want) {
					t.Fatalf("distancias de %q = %v, esperaba %v (todas: %v)", name, got[name], want, got)
				}
			}
		})
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestResolveErrors(t *testing.T) {
	cases := []struct {
		name    string
		source  string
		wantMsg string
	}{
		{"return en el nivel global", "return 1;", "no se puede usar 'return' fuera de una función"},
		{"return dentro de un bloque global", "{ return; }", "no se puede usar 'return' fuera de una función"},
		{"return dentro de un while global", "while (true) { return; }", "no se puede usar 'return' fuera de una función"},
		{"leer una local en su propio inicializador", "{ var a = a; }", "no se puede leer una variable local en su propio inicializador"},
		{
			// Aunque exista una `a` en el bloque de afuera, la interna ya está
			// declarada (sin definir) en su scope cuando se lee el inicializador.
			"el inicializador no ve la variable de afuera con el mismo nombre",
			"{ var a = 1; { var a = a; } }",
			"no se puede leer una variable local en su propio inicializador",
		},
		{"redeclarar en el mismo bloque", "{ var a = 1; var a = 2; }", "ya existe una variable con este nombre en este scope"},
		{"parámetro duplicado", "fun f(a, a) {}", "ya existe una variable con este nombre en este scope"},
		{"redeclarar un parámetro en el cuerpo", "fun f(a) { var a = 1; }", "ya existe una variable con este nombre en este scope"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, errs := resolver.Resolve(parse(t, c.source))
			if len(errs) != 1 {
				t.Fatalf("esperaba exactamente 1 error, obtuve %d: %v", len(errs), errs)
			}
			if !strings.Contains(errs[0].Error(), c.wantMsg) {
				t.Fatalf("error = %q, esperaba que contenga %q", errs[0], c.wantMsg)
			}
		})
	}
}

func TestResolveAllowedCases(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{"redeclarar en el scope global está permitido", "var a = 1; var a = 2;"},
		{"una global puede usarse en su propio inicializador", "var a = a;"},
		{"shadowing en un bloque interno está permitido", "{ var a = 1; { var a = 2; print a; } }"},
		{"return dentro de una función anidada", "fun f() { fun g() { return 1; } return g(); }"},
		{"una función local puede ser recursiva", "{ fun f(n) { if (n > 0) f(n - 1); } }"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, errs := resolver.Resolve(parse(t, c.source)); len(errs) > 0 {
				t.Fatalf("no esperaba errores en %q, obtuve: %v", c.source, errs)
			}
		})
	}
}

func TestResolveReportsAllErrors(t *testing.T) {
	source := `
		return 1;
		{ var a = 1; var a = 2; }
		{ var b = b; }
	`
	_, errs := resolver.Resolve(parse(t, source))
	if len(errs) != 3 {
		t.Fatalf("esperaba 3 errores acumulados, obtuve %d: %v", len(errs), errs)
	}
}
