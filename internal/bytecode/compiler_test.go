package bytecode_test

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/federicohonda/glox-110766-110759/internal/bytecode"
	"github.com/federicohonda/glox-110766-110759/internal/parser"
	"github.com/federicohonda/glox-110766-110759/internal/scanner"
)

// compileSource escanea, parsea y compila una expresión.
func compileSource(t *testing.T, source string) (*bytecode.Chunk, error) {
	t.Helper()
	tokens, errs := scanner.New(source).Scan()
	if len(errs) > 0 {
		t.Fatalf("error léxico inesperado al escanear %q: %v", source, errs)
	}
	expr, err := parser.New(tokens).ParseExpression()
	if err != nil {
		t.Fatalf("error sintáctico inesperado al parsear %q: %v", source, err)
	}
	return bytecode.Compile(expr)
}

func mustCompile(t *testing.T, source string) *bytecode.Chunk {
	t.Helper()
	chunk, err := compileSource(t, source)
	if err != nil {
		t.Fatalf("Compile(%q) falló con error: %v", source, err)
	}
	return chunk
}

// instructions resume un chunk en una sola línea, sin offsets ni líneas del
// fuente: el nombre de cada instrucción y, para OpConstant, el valor que apila
// (ej. "OP_CONSTANT 2, OP_NEGATE, OP_RETURN").
func instructions(chunk *bytecode.Chunk) string {
	var parts []string
	for offset := 0; offset < len(chunk.Code); offset++ {
		op := bytecode.OpCode(chunk.Code[offset])
		if op == bytecode.OpConstant {
			offset++
			parts = append(parts, fmt.Sprintf("%s %v", op, chunk.Constants[chunk.Code[offset]]))
		} else {
			parts = append(parts, op.String())
		}
	}
	return strings.Join(parts, ", ")
}

// El checkpoint de la parte: en `2 + 3 * 4` la multiplicación se emite antes
// que la suma, y eso se ve en el listado del disassembler.
func TestCompilePrecedenciaEnElDisassembler(t *testing.T) {
	chunk := mustCompile(t, "2 + 3 * 4")

	expected := strings.Join([]string{
		"== 2 + 3 * 4 ==",
		"0000    1 OP_CONSTANT         0 '2'",
		"0002    | OP_CONSTANT         1 '3'",
		"0004    | OP_CONSTANT         2 '4'",
		"0006    | OP_MULTIPLY",
		"0007    | OP_ADD",
		"0008    | OP_RETURN",
		"",
	}, "\n")

	if got := chunk.Disassemble("2 + 3 * 4"); got != expected {
		t.Errorf("Disassemble distinto\n--- obtenido\n%s--- esperado\n%s", got, expected)
	}
}

func TestCompileExpresionesAritmeticas(t *testing.T) {
	tests := []struct {
		source   string
		expected string
	}{
		{"1", "OP_CONSTANT 1, OP_RETURN"},
		{"1 + 2", "OP_CONSTANT 1, OP_CONSTANT 2, OP_ADD, OP_RETURN"},
		{"1 - 2", "OP_CONSTANT 1, OP_CONSTANT 2, OP_SUBTRACT, OP_RETURN"},
		{"1 * 2", "OP_CONSTANT 1, OP_CONSTANT 2, OP_MULTIPLY, OP_RETURN"},
		{"1 / 2", "OP_CONSTANT 1, OP_CONSTANT 2, OP_DIVIDE, OP_RETURN"},
		{"1 % 2", "OP_CONSTANT 1, OP_CONSTANT 2, OP_MODULO, OP_RETURN"},
		{"-5", "OP_CONSTANT 5, OP_NEGATE, OP_RETURN"},
		{"--5", "OP_CONSTANT 5, OP_NEGATE, OP_NEGATE, OP_RETURN"},

		// Precedencia: * antes que +, de los dos lados.
		{"2 + 3 * 4", "OP_CONSTANT 2, OP_CONSTANT 3, OP_CONSTANT 4, OP_MULTIPLY, OP_ADD, OP_RETURN"},
		{"2 * 3 + 4", "OP_CONSTANT 2, OP_CONSTANT 3, OP_MULTIPLY, OP_CONSTANT 4, OP_ADD, OP_RETURN"},

		// Los paréntesis cambian el orden pero no emiten nada.
		{"(2 + 3) * 4", "OP_CONSTANT 2, OP_CONSTANT 3, OP_ADD, OP_CONSTANT 4, OP_MULTIPLY, OP_RETURN"},
		{"(1)", "OP_CONSTANT 1, OP_RETURN"},

		// Asociatividad a izquierda: (1 - 2) - 3, no 1 - (2 - 3).
		{"1 - 2 - 3", "OP_CONSTANT 1, OP_CONSTANT 2, OP_SUBTRACT, OP_CONSTANT 3, OP_SUBTRACT, OP_RETURN"},
		{"8 / 4 / 2", "OP_CONSTANT 8, OP_CONSTANT 4, OP_DIVIDE, OP_CONSTANT 2, OP_DIVIDE, OP_RETURN"},

		// El menos unario liga más fuerte que cualquier binario.
		{"-1 - -2", "OP_CONSTANT 1, OP_NEGATE, OP_CONSTANT 2, OP_NEGATE, OP_SUBTRACT, OP_RETURN"},
		{"-(1 + 2)", "OP_CONSTANT 1, OP_CONSTANT 2, OP_ADD, OP_NEGATE, OP_RETURN"},
		{"-2 * 3", "OP_CONSTANT 2, OP_NEGATE, OP_CONSTANT 3, OP_MULTIPLY, OP_RETURN"},

		// Cualquier literal va a la tabla de constantes, no solo los números.
		{`"a" + "b"`, "OP_CONSTANT a, OP_CONSTANT b, OP_ADD, OP_RETURN"},
		{"true", "OP_CONSTANT true, OP_RETURN"},
		{"nil", "OP_CONSTANT <nil>, OP_RETURN"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := instructions(mustCompile(t, tt.source)); got != tt.expected {
				t.Errorf("Compile(%q)\n obtenido: %s\n esperado: %s", tt.source, got, tt.expected)
			}
		})
	}
}

// Cada instrucción queda anotada con la línea de su propio token: la del
// literal para las constantes y la del operador para las operaciones. Como se
// emite en postorden, las líneas no quedan en orden creciente.
func TestCompileAnotaLaLineaDeCadaInstruccion(t *testing.T) {
	chunk := mustCompile(t, "1 +\n2 *\n3")

	expected := strings.Join([]string{
		"== líneas ==",
		"0000    1 OP_CONSTANT         0 '1'",
		"0002    2 OP_CONSTANT         1 '2'",
		"0004    3 OP_CONSTANT         2 '3'",
		"0006    2 OP_MULTIPLY",
		"0007    1 OP_ADD",
		"0008    | OP_RETURN",
		"",
	}, "\n")

	if got := chunk.Disassemble("líneas"); got != expected {
		t.Errorf("Disassemble distinto\n--- obtenido\n%s--- esperado\n%s", got, expected)
	}
	if len(chunk.Lines) != len(chunk.Code) {
		t.Errorf("len(Lines) = %d, len(Code) = %d: tienen que ser paralelos", len(chunk.Lines), len(chunk.Code))
	}
}

// Cada literal ocupa su propio lugar en la tabla, en el orden en que aparece.
func TestCompileTablaDeConstantes(t *testing.T) {
	chunk := mustCompile(t, "1 + 2 + 1")

	expected := []bytecode.Value{1.0, 2.0, 1.0}
	if !slices.Equal(chunk.Constants, expected) {
		t.Errorf("Constants = %v, esperado %v", chunk.Constants, expected)
	}
}

// sumOf arma una suma de n literales, cada uno en su propia línea.
func sumOf(n int) string {
	terms := make([]string, n)
	for i := range terms {
		terms[i] = "1"
	}
	return strings.Join(terms, " +\n")
}

func TestCompileLimiteDeConstantes(t *testing.T) {
	t.Run("256 constantes entran", func(t *testing.T) {
		chunk := mustCompile(t, sumOf(256))
		if len(chunk.Constants) != 256 {
			t.Errorf("len(Constants) = %d, esperado 256", len(chunk.Constants))
		}
	})

	t.Run("257 es un error de compilación", func(t *testing.T) {
		chunk, err := compileSource(t, sumOf(257))
		if chunk != nil {
			t.Error("se esperaba un chunk nil junto con el error")
		}
		var compileErr *bytecode.CompileError
		if !errors.As(err, &compileErr) {
			t.Fatalf("se esperaba un *CompileError, se obtuvo: %v", err)
		}
		// El error apunta al literal que ya no entra, que está en la línea 257.
		if compileErr.Line != 257 {
			t.Errorf("Line = %d, esperado 257", compileErr.Line)
		}
		expected := "[línea 257] Error de compilación: demasiadas constantes en un mismo chunk (el máximo es 256)."
		if err.Error() != expected {
			t.Errorf("mensaje = %q, esperado %q", err.Error(), expected)
		}
	})
}

// Lo que el lenguaje tiene pero el compilador todavía no traduce se rechaza
// con un error, en vez de generar bytecode incompleto.
func TestCompileConstruccionesTodaviaNoSoportadas(t *testing.T) {
	tests := []struct {
		source   string
		expected string
	}{
		{"!true", "[línea 1] Error de compilación: el compilador a bytecode todavía no soporta el operador '!'."},
		{"1 < 2", "[línea 1] Error de compilación: el compilador a bytecode todavía no soporta el operador '<'."},
		{"1 == 1", "[línea 1] Error de compilación: el compilador a bytecode todavía no soporta el operador '=='."},
		{"true and false", "[línea 1] Error de compilación: el compilador a bytecode todavía no soporta el operador 'and'."},
		{"x", "[línea 1] Error de compilación: el compilador a bytecode todavía no soporta las variables."},
		{"x = 1", "[línea 1] Error de compilación: el compilador a bytecode todavía no soporta la asignación."},
		{"f(1)", "[línea 1] Error de compilación: el compilador a bytecode todavía no soporta las llamadas."},
		// El error puede estar en cualquier profundidad del árbol.
		{"1 + 2 *\n(3 - x)", "[línea 2] Error de compilación: el compilador a bytecode todavía no soporta las variables."},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			chunk, err := compileSource(t, tt.source)
			if chunk != nil {
				t.Error("se esperaba un chunk nil junto con el error")
			}
			if err == nil {
				t.Fatalf("Compile(%q) no devolvió error", tt.source)
			}
			if err.Error() != tt.expected {
				t.Errorf("mensaje = %q, esperado %q", err.Error(), tt.expected)
			}
		})
	}
}
