package bytecode_test

import (
	"io"
	"testing"

	"github.com/federicohonda/glox-110766-110759/internal/bytecode"
	"github.com/federicohonda/glox-110766-110759/internal/parser"
	"github.com/federicohonda/glox-110766-110759/internal/resolver"
	"github.com/federicohonda/glox-110766-110759/internal/scanner"
)

// benchmarkVM prepara el chunk una sola vez y mide la ejecución pura de la VM.
func benchmarkVM(b *testing.B, source string) {
	b.Helper()
	tokens, errs := scanner.New(source).Scan()
	if len(errs) > 0 {
		b.Fatalf("error léxico: %v", errs)
	}
	stmts, parseErrs := parser.New(tokens).Parse()
	if len(parseErrs) > 0 {
		b.Fatalf("error sintáctico: %v", parseErrs)
	}
	_, resolveErrs := resolver.Resolve(stmts)
	if len(resolveErrs) > 0 {
		b.Fatalf("error semántico: %v", resolveErrs)
	}
	chunk, err := bytecode.CompileProgram(stmts)
	if err != nil {
		b.Fatalf("error de compilación: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vm := bytecode.NewVM(io.Discard)
		if _, err := vm.Run(chunk); err != nil {
			b.Fatalf("error de ejecución en VM: %v", err)
		}
	}
}

// BenchmarkVMFibRecursive mide el costo de llamadas a funciones recursivas y gestión de frames en la VM.
// Mismo caso de prueba que en el interpreter_bench_test.go para comparar directamente.
func BenchmarkVMFibRecursive(b *testing.B) {
	source := `
fun fib(n) {
    if (n < 2) return n;
    return fib(n - 1) + fib(n - 2);
}
fib(15);
`
	benchmarkVM(b, source)
}

// BenchmarkVMLoopIntensive mide el costo de iteración sobre bucles, aritmética y asignación en la VM.
func BenchmarkVMLoopIntensive(b *testing.B) {
	source := `
var sum = 0;
for (var i = 0; i < 10000; i = i + 1) {
    sum = sum + i;
}
`
	benchmarkVM(b, source)
}

// BenchmarkVMClosureCounter mide el costo de creación y captura léxica de entornos con closures en la VM.
func BenchmarkVMClosureCounter(b *testing.B) {
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
	benchmarkVM(b, source)
}

// BenchmarkVMCalls mide llamadas a funciones no recursivas en la VM.
func BenchmarkVMCalls(b *testing.B) {
	source := `
fun sumar(a, b) {
    return a + b;
}
var res = 0;
for (var i = 0; i < 10000; i = i + 1) {
    res = sumar(res, i);
}
`
	benchmarkVM(b, source)
}

// BenchmarkVMPipelineEndToEnd mide el costo de atravesar el pipeline completo: scan -> parse -> resolve -> compile -> vm.Run.
func BenchmarkVMPipelineEndToEnd(b *testing.B) {
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
		_, _ = resolver.Resolve(stmts)
		chunk, _ := bytecode.CompileProgram(stmts)
		vm := bytecode.NewVM(io.Discard)
		_, _ = vm.Run(chunk)
	}
}
