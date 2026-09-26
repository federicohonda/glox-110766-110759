package scanner

import (
	"strings"
	"testing"
)

// BenchmarkScanArithmetic mide la velocidad de escaneo sobre expresiones aritméticas repetitivas.
func BenchmarkScanArithmetic(b *testing.B) {
	source := strings.Repeat("1 + 2 * 3 - 4 / 5;\n", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		New(source).Scan()
	}
}

// BenchmarkScanKeywords mide la velocidad de escaneo sobre palabras clave, identificadores, strings y números.
func BenchmarkScanKeywords(b *testing.B) {
	source := strings.Repeat("var x = 123; if (x > 0) { print \"hello\"; } while (true) { var y = nil; }\n", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		New(source).Scan()
	}
}
