package parser_test

import (
	"testing"

	"github.com/federicohonda/glox-110766-110759/internal/parser"
	"github.com/federicohonda/glox-110766-110759/internal/scanner"
)

// parseTree escanea y parsea una expresión y devuelve su representación en
// notación prefija parentizada (ver ast.Expr.String), que es lo que también
// imprime el modo --parsing del CLI.
func parseTree(t *testing.T, source string) string {
	t.Helper()
	tokens, scanErrs := scanner.New(source).Scan()
	if len(scanErrs) > 0 {
		t.Fatalf("error de escaneo inesperado en %q: %v", source, scanErrs)
	}
	expr, err := parser.New(tokens).Parse()
	if err != nil {
		t.Fatalf("error de parseo inesperado en %q: %v", source, err)
	}
	return expr.String()
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

func TestParserReportsSyntaxErrors(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{"expresión incompleta tras operador binario", "1 + "},
		{"paréntesis sin cerrar", "(1 + 2"},
		{"token inesperado en primary", ";"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tokens, scanErrs := scanner.New(c.source).Scan()
			if len(scanErrs) > 0 {
				t.Fatalf("error de escaneo inesperado en %q: %v", c.source, scanErrs)
			}
			if _, err := parser.New(tokens).Parse(); err == nil {
				t.Fatalf("esperaba un error de sintaxis parseando %q, no obtuve ninguno", c.source)
			}
		})
	}
}
