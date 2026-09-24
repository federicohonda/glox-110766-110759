package interpreter_test

import (
	"testing"

	"github.com/federicohonda/glox-110766-110759/internal/interpreter"
	"github.com/federicohonda/glox-110766-110759/internal/parser"
	"github.com/federicohonda/glox-110766-110759/internal/resolver"
	"github.com/federicohonda/glox-110766-110759/internal/scanner"
)

// benchmarkInterpreter prepara el AST y resolución una sola vez, y mide la ejecución pura del intérprete.
func benchmarkInterpreter(b *testing.B, source string) {
	b.Helper()
	tokens, errs := scanner.New(source).Scan()
	if len(errs) > 0 {
		b.Fatalf("error léxico: %v", errs)
	}
	stmts, parseErrs := parser.New(tokens).Parse()
	if len(parseErrs) > 0 {
		b.Fatalf("error sintáctico: %v", parseErrs)
	}
	locals, resolveErrs := resolver.Resolve(stmts)
	if len(resolveErrs) > 0 {
		b.Fatalf("error semántico: %v", resolveErrs)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		interp := interpreter.New()
		interp.Resolve(locals)
		if err := interp.Interpret(stmts); err != nil {
			b.Fatalf("error de ejecución: %v", err)
		}
	}
}

// BenchmarkFibRecursive mide el costo de llamadas a funciones recursivas y creación de entornos/frames.
func BenchmarkFibRecursive(b *testing.B) {
	source := `
fun fib(n) {
    if (n < 2) return n;
    return fib(n - 1) + fib(n - 2);
}
fib(15);
`
	benchmarkInterpreter(b, source)
}

// BenchmarkLoopIntensive mide el costo de iteración sobre bucles y operaciones aritméticas en variables locales.
func BenchmarkLoopIntensive(b *testing.B) {
	source := `
var sum = 0;
for (var i = 0; i < 10000; i = i + 1) {
    sum = sum + i;
}
`
	benchmarkInterpreter(b, source)
}

// BenchmarkClosureCounter mide el costo de creación y captura léxica de entornos con closures.
func BenchmarkClosureCounter(b *testing.B) {
	source := `
fun makeCounter() {
    var count = 0;
    fun increment() {
        count = count + 1;
        return count;
    }
    return increment;
}
var counter = makeCounter();
for (var i = 0; i < 1000; i = i + 1) {
    counter();
}
`
	benchmarkInterpreter(b, source)
}

// BenchmarkPipelineEndToEnd mide el costo de atravesar el pipeline completo: scan -> parse -> resolve -> interpret.
func BenchmarkPipelineEndToEnd(b *testing.B) {
	source := `
fun suma(a, b) {
    return a + b;
}
var res = 0;
for (var i = 0; i < 100; i = i + 1) {
    res = suma(res, i);
}
`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tokens, _ := scanner.New(source).Scan()
		stmts, _ := parser.New(tokens).Parse()
		locals, _ := resolver.Resolve(stmts)
		interp := interpreter.New()
		interp.Resolve(locals)
		_ = interp.Interpret(stmts)
	}
}
