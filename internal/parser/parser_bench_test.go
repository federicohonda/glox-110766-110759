package parser_test

import (
	"strings"
	"testing"

	"github.com/federicohonda/glox-110766-110759/internal/parser"
	"github.com/federicohonda/glox-110766-110759/internal/scanner"
)

// BenchmarkParseExpressions mide el rendimiento del parser construyendo árboles de expresiones anidadas.
func BenchmarkParseExpressions(b *testing.B) {
	source := strings.Repeat("var x = 1 + 2 * 3 / (4 - 5) >= 6 == true;\n", 100)
	tokens, _ := scanner.New(source).Scan()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.New(tokens).Parse()
	}
}

// BenchmarkParseFunctions mide el rendimiento del parser analizando declaraciones y bloques de funciones.
func BenchmarkParseFunctions(b *testing.B) {
	source := strings.Repeat("fun f(a, b) { var c = a + b; if (c > 0) { print c; } return c; }\n", 50)
	tokens, _ := scanner.New(source).Scan()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.New(tokens).Parse()
	}
}
