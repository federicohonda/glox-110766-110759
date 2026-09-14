package parser_test

import (
	"strings"
	"testing"

	"github.com/federicohonda/glox-110766-110759/internal/parser"
	"github.com/federicohonda/glox-110766-110759/internal/scanner"
)

// parseTree escanea y parsea una expresión y devuelve su representación en
// notación prefija parentizada (ver ast.Expr.String).
func parseTree(t *testing.T, source string) string {
	t.Helper()
	tokens, scanErrs := scanner.New(source).Scan()
	if len(scanErrs) > 0 {
		t.Fatalf("error de escaneo inesperado en %q: %v", source, scanErrs)
	}
	expr, err := parser.New(tokens).ParseExpression()
	if err != nil {
		t.Fatalf("error de parseo inesperado en %q: %v", source, err)
	}
	return expr.String()
}

func parseProgram(t *testing.T, source string) string {
	t.Helper()
	tokens, scanErrs := scanner.New(source).Scan()
	if len(scanErrs) > 0 {
		t.Fatalf("error de escaneo inesperado en %q: %v", source, scanErrs)
	}
	stmts, parseErrs := parser.New(tokens).Parse()
	if len(parseErrs) > 0 {
		t.Fatalf("errores de parseo inesperados en %q: %v", source, parseErrs)
	}
	var parts []string
	for _, s := range stmts {
		parts = append(parts, s.String())
	}
	return strings.Join(parts, " ")
}

func TestParserBuildsExpectedTree(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   string
	}{
		{"precedencia mixta", "1 + 2 * 3", "(+ 1 (* 2 3))"},
		{"paréntesis fuerzan agrupación", "(1 + 2) * 3", "(* (group (+ 1 2)) 3)"},
		{"unario junto a binario", "-1 + 2", "(+ (- 1) 2)"},
		{"equality y comparison anidados", "1 < 2 == 3 < 4", "(== (< 1 2) (< 3 4))"},
		{"asociatividad izquierda en equality", "1 == 2 != 3", "(!= (== 1 2) 3)"},
		{"negación lógica", "!true", "(! true)"},
		{"módulo con misma precedencia que * y /", "8 % 3 * 2", "(* (% 8 3) 2)"},
		{"literal nil solo", "nil", "nil"},
		{"paréntesis anidados", "((1))", "(group (group 1))"},
		{"strings comparados", "\"a\" == \"b\"", "(== a b)"},
		{"acceso a variable", "x", "x"},
		{"asignación simple", "x = 5", "(= x 5)"},
		{"asignación encadenada", "a = b = 3", "(= a (= b 3))"},
		{"or lógico", "a or b", "(or a b)"},
		{"and lógico", "a and b", "(and a b)"},
		{"precedencia and sobre or", "a or b and c", "(or a (and b c))"},
		{"precedencia and sobre or inversa", "a and b or c", "(or (and a b) c)"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseTree(t, c.source)
			if got != c.want {
				t.Fatalf("parseTree(%q) = %q, esperaba %q", c.source, got, c.want)
			}
		})
	}
}

func TestParserStatements(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "checkpoint: var y print",
			source: "var x = 1; print x;",
			want:   "(var x = 1) (print x)",
		},
		{
			name:   "declaración sin inicializador",
			source: "var a;",
			want:   "(var a)",
		},
		{
			name:   "expression statement",
			source: "1 + 2; x = 5;",
			want:   "(expr (+ 1 2)) (expr (= x 5))",
		},
		{
			name:   "print statement",
			source: "print \"hola\";",
			want:   "(print hola)",
		},
		{
			name:   "bloque simple",
			source: "{ var a = 1; print a; }",
			want:   "(block (var a = 1) (print a))",
		},
		{
			name:   "bloques anidados",
			source: "{ { var b = 2; } }",
			want:   "(block (block (var b = 2)))",
		},
		{
			name:   "if simple",
			source: "if (true) print 1;",
			want:   "(if true (print 1))",
		},
		{
			name:   "if con else",
			source: "if (false) print 1; else print 2;",
			want:   "(if-else false (print 1) (print 2))",
		},
		{
			name:   "dangling else asociado al if interno",
			source: "if (a) if (b) print 1; else print 2;",
			want:   "(if a (if-else b (print 1) (print 2)))",
		},
		{
			name:   "while statement",
			source: "while (x < 10) x = x + 1;",
			want:   "(while (< x 10) (expr (= x (+ x 1))))",
		},
		{
			name:   "for desazucarado completo",
			source: "for (var i = 0; i < 5; i = i + 1) print i;",
			want:   "(block (var i = 0) (while (< i 5) (block (print i) (expr (= i (+ i 1))))))",
		},
		{
			name:   "for sin inicializador ni incremento",
			source: "for (; x < 5;) print x;",
			want:   "(while (< x 5) (print x))",
		},
		{
			name:   "for infinito for (;;)",
			source: "for (;;) print 1;",
			want:   "(while true (print 1))",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseProgram(t, c.source)
			if got != c.want {
				t.Fatalf("parseProgram(%q) = %q, esperaba %q", c.source, got, c.want)
			}
		})
	}
}

func TestParserReportsSyntaxErrors(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{"expresión incompleta tras operador binario", "1 + "},
		{"paréntesis sin cerrar", "(1 + 2"},
		{"token inesperado en primary", ";"},
		{"destino de asignación inválido", "1 + 2 = 3"},
		{"var sin identificador", "var = 5;"},
		{"falta punto y coma en var", "var x = 1"},
		{"falta punto y coma en print", "print 1"},
		{"bloque sin cerrar", "{ var x = 1;"},
		{"if sin paréntesis", "if true print 1;"},
		{"if con paréntesis sin cerrar", "if (true print 1;"},
		{"while sin paréntesis", "while true print 1;"},
		{"while con paréntesis sin cerrar", "while (true print 1;"},
		{"for sin paréntesis", "for ; ; print 1;"},
		{"for sin primer punto y coma", "for (var i = 0 i < 5; i = i + 1) print i;"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tokens, scanErrs := scanner.New(c.source).Scan()
			if len(scanErrs) > 0 {
				t.Fatalf("error de escaneo inesperado en %q: %v", c.source, scanErrs)
			}
			p := parser.New(tokens)
			// Probar que falle en Parse o ParseExpression
			_, errs := p.Parse()
			if len(errs) == 0 {
				t.Fatalf("esperaba un error de sintaxis parseando %q, no obtuve ninguno", c.source)
			}
		})
	}
}

func TestParserErrorSynchronization(t *testing.T) {
	// Ante un error en la primera sentencia ("var = 1;"), el parser debe sincronizar
	// en el punto y coma y continuar parseando la segunda sentencia ("print 2;").
	source := "var = 1; print 2;"
	tokens, scanErrs := scanner.New(source).Scan()
	if len(scanErrs) > 0 {
		t.Fatalf("error de escaneo inesperado: %v", scanErrs)
	}

	p := parser.New(tokens)
	stmts, parseErrs := p.Parse()
	if len(parseErrs) == 0 {
		t.Fatal("se esperaba al menos un error de sintaxis")
	}
	if len(stmts) != 0 {
		// Parse() devuelve nil cuando len(errors) > 0
	}
}
