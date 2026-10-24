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
		if op == bytecode.OpConstant || op == bytecode.OpDefineGlobal || op == bytecode.OpGetGlobal || op == bytecode.OpSetGlobal {
			offset++
			parts = append(parts, fmt.Sprintf("%s %v", op, chunk.Constants[chunk.Code[offset]]))
		} else if op == bytecode.OpGetLocal || op == bytecode.OpSetLocal {
			offset++
			parts = append(parts, fmt.Sprintf("%s %d", op, chunk.Code[offset]))
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
		{"f(1)", "[línea 1] Error de compilación: el compilador a bytecode todavía no soporta las llamadas."},
		{"-(1 < 2)", "[línea 1] Error de compilación: el compilador a bytecode todavía no soporta el operador '<'."},
		{"(1 + 2) == 3", "[línea 1] Error de compilación: el compilador a bytecode todavía no soporta el operador '=='."},
		{"f(1) + 2", "[línea 1] Error de compilación: el compilador a bytecode todavía no soporta las llamadas."},
		// El resto de los operadores que faltan.
		{"1 > 2", "[línea 1] Error de compilación: el compilador a bytecode todavía no soporta el operador '>'."},
		{"1 >= 2", "[línea 1] Error de compilación: el compilador a bytecode todavía no soporta el operador '>='."},
		{"1 <= 2", "[línea 1] Error de compilación: el compilador a bytecode todavía no soporta el operador '<='."},
		{"1 != 2", "[línea 1] Error de compilación: el compilador a bytecode todavía no soporta el operador '!='."},
		{"nil or 1", "[línea 1] Error de compilación: el compilador a bytecode todavía no soporta el operador 'or'."},
		{"!!true", "[línea 1] Error de compilación: el compilador a bytecode todavía no soporta el operador '!'."},
		// Se corta en el primer error: de los dos problemas, se reporta el de
		// la línea 1.
		{"f(1) +\ny()", "[línea 1] Error de compilación: el compilador a bytecode todavía no soporta las llamadas."},
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

// Precedencia con paréntesis anidados: lo que está más adentro se emite
// primero, sin importar en qué orden está escrito.
func TestCompileParentesisAnidados(t *testing.T) {
	tests := []struct {
		source   string
		expected string
	}{
		{
			"((1 + 2) * (3 - (8 / (6 % 4))))",
			"OP_CONSTANT 1, OP_CONSTANT 2, OP_ADD, OP_CONSTANT 3, OP_CONSTANT 8, OP_CONSTANT 6, OP_CONSTANT 4, OP_MODULO, OP_DIVIDE, OP_SUBTRACT, OP_MULTIPLY, OP_RETURN",
		},
		{
			// Anidado hacia la derecha: se apilan todos los operandos antes
			// de la primera suma.
			"1 + (2 + (3 + (4 + 5)))",
			"OP_CONSTANT 1, OP_CONSTANT 2, OP_CONSTANT 3, OP_CONSTANT 4, OP_CONSTANT 5, OP_ADD, OP_ADD, OP_ADD, OP_ADD, OP_RETURN",
		},
		{
			// Anidado hacia la izquierda: nunca hay más de dos valores apilados.
			"(((1 + 2) + 3) + 4) + 5",
			"OP_CONSTANT 1, OP_CONSTANT 2, OP_ADD, OP_CONSTANT 3, OP_ADD, OP_CONSTANT 4, OP_ADD, OP_CONSTANT 5, OP_ADD, OP_RETURN",
		},
		{
			"2 * (3 + 4) * 5",
			"OP_CONSTANT 2, OP_CONSTANT 3, OP_CONSTANT 4, OP_ADD, OP_MULTIPLY, OP_CONSTANT 5, OP_MULTIPLY, OP_RETURN",
		},
		{
			"-(-(-(1)))",
			"OP_CONSTANT 1, OP_NEGATE, OP_NEGATE, OP_NEGATE, OP_RETURN",
		},
		{
			"-((1 + 2) * -(3 - 4))",
			"OP_CONSTANT 1, OP_CONSTANT 2, OP_ADD, OP_CONSTANT 3, OP_CONSTANT 4, OP_SUBTRACT, OP_NEGATE, OP_MULTIPLY, OP_NEGATE, OP_RETURN",
		},
		{
			"(1 + 2) % (3 * 4)",
			"OP_CONSTANT 1, OP_CONSTANT 2, OP_ADD, OP_CONSTANT 3, OP_CONSTANT 4, OP_MULTIPLY, OP_MODULO, OP_RETURN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := instructions(mustCompile(t, tt.source)); got != tt.expected {
				t.Errorf("Compile(%q)\n obtenido: %s\n esperado: %s", tt.source, got, tt.expected)
			}
		})
	}
}

// Los paréntesis que no cambian el orden no dejan rastro en el bytecode, por
// más niveles que haya.
func TestCompileParentesisRedundantes(t *testing.T) {
	expected := mustCompile(t, "1 + 2 * 3")

	sources := []string{
		"(1 + 2 * 3)",
		"1 + (2 * 3)",
		"(1) + ((2) * (3))",
		"((((1)))) + ((((2 * 3))))",
		strings.Repeat("(", 500) + "1 + 2 * 3" + strings.Repeat(")", 500),
	}

	for i, source := range sources {
		t.Run(fmt.Sprintf("caso %d", i), func(t *testing.T) {
			chunk := mustCompile(t, source)
			if !slices.Equal(chunk.Code, expected.Code) {
				t.Errorf("Code = %v, esperado %v", chunk.Code, expected.Code)
			}
			if !slices.Equal(chunk.Constants, expected.Constants) {
				t.Errorf("Constants = %v, esperado %v", chunk.Constants, expected.Constants)
			}
		})
	}
}

// La última constante que entra en la tabla usa el índice 255, el más grande
// que se puede escribir en un byte.
func TestCompileUltimoIndiceDeConstante(t *testing.T) {
	chunk := mustCompile(t, sumOf(256))

	// El chunk termina en: OP_CONSTANT 255, OP_ADD, OP_RETURN.
	tail := chunk.Code[len(chunk.Code)-4:]
	expected := []byte{byte(bytecode.OpConstant), 255, byte(bytecode.OpAdd), byte(bytecode.OpReturn)}
	if !slices.Equal(tail, expected) {
		t.Errorf("final del chunk = %v, esperado %v", tail, expected)
	}

	listing := chunk.Disassemble("lleno")
	if !strings.Contains(listing, "OP_CONSTANT       255 '1'") {
		t.Errorf("el listado no muestra la constante 255:\n%s", listing[len(listing)-200:])
	}
}

func TestCompileSinExpresion(t *testing.T) {
	chunk, err := bytecode.Compile(nil)
	if chunk != nil {
		t.Error("se esperaba un chunk nil junto con el error")
	}
	if err == nil {
		t.Fatal("Compile(nil) no devolvió error")
	}
}

func TestCompileProgramSentenciasTodaviaNoSoportadas(t *testing.T) {
	tests := []struct {
		source   string
		expected string
	}{
		{"if (true) print 1;", "[línea 1] Error de compilación: el compilador a bytecode todavía no soporta la sentencia 'if'."},
		{"while (true) print 1;", "[línea 1] Error de compilación: el compilador a bytecode todavía no soporta el bucle 'while'."},
		{"fun f() {}", "[línea 1] Error de compilación: el compilador a bytecode todavía no soporta las funciones."},
		{"return 1;", "[línea 1] Error de compilación: el compilador a bytecode todavía no soporta la sentencia 'return'."},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			chunk, err := compileProgramSource(t, tt.source)
			if chunk != nil {
				t.Error("se esperaba un chunk nil junto con el error")
			}
			if err == nil {
				t.Fatalf("CompileProgram(%q) no devolvió error", tt.source)
			}
			if err.Error() != tt.expected {
				t.Errorf("mensaje = %q, esperado %q", err.Error(), tt.expected)
			}
		})
	}
}

func compileProgramSource(t *testing.T, source string) (*bytecode.Chunk, error) {
	t.Helper()
	tokens, errs := scanner.New(source).Scan()
	if len(errs) > 0 {
		t.Fatalf("error léxico inesperado al escanear %q: %v", source, errs)
	}
	stmts, parseErrs := parser.New(tokens).Parse()
	if len(parseErrs) > 0 {
		t.Fatalf("error sintáctico inesperado al parsear %q: %v", source, parseErrs)
	}
	return bytecode.CompileProgram(stmts)
}

func mustCompileProgram(t *testing.T, source string) *bytecode.Chunk {
	t.Helper()
	chunk, err := compileProgramSource(t, source)
	if err != nil {
		t.Fatalf("CompileProgram(%q) falló con error: %v", source, err)
	}
	return chunk
}

func TestCompileProgramGlobalesYPrint(t *testing.T) {
	chunk := mustCompileProgram(t, "var x = 1; print x;")
	expected := "OP_CONSTANT 1, OP_DEFINE_GLOBAL x, OP_GET_GLOBAL x, OP_PRINT, OP_RETURN"
	if got := instructions(chunk); got != expected {
		t.Errorf("instrucciones = %q, esperado %q", got, expected)
	}

	chunkNil := mustCompileProgram(t, "var y;")
	expectedNil := "OP_CONSTANT <nil>, OP_DEFINE_GLOBAL y, OP_RETURN"
	if got := instructions(chunkNil); got != expectedNil {
		t.Errorf("instrucciones = %q, esperado %q", got, expectedNil)
	}
}

func TestCompileProgramAsignacionComoExpresion(t *testing.T) {
	chunk := mustCompileProgram(t, "var a; var b; a = b = 3;")
	expected := "OP_CONSTANT <nil>, OP_DEFINE_GLOBAL a, OP_CONSTANT <nil>, OP_DEFINE_GLOBAL b, OP_CONSTANT 3, OP_SET_GLOBAL b, OP_SET_GLOBAL a, OP_POP, OP_RETURN"
	if got := instructions(chunk); got != expected {
		t.Errorf("instrucciones = %q, esperado %q", got, expected)
	}
}

func TestCompileProgramDeduplicaNombresDeGlobales(t *testing.T) {
	chunk := mustCompileProgram(t, "var x = 1; x = x + 1; print x;")
	// "x" debe aparecer solo una vez en Constants.
	countX := 0
	for _, c := range chunk.Constants {
		if c == "x" {
			countX++
		}
	}
	if countX != 1 {
		t.Errorf("nombre 'x' aparece %d veces en constantes (esperado 1): %v", countX, chunk.Constants)
	}
}

func TestCompileProgramLocales(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{
			name:     "declaracion y uso",
			source:   "{ var a = 1; var b = 2; print a + b; }",
			expected: "OP_CONSTANT 1, OP_CONSTANT 2, OP_GET_LOCAL 0, OP_GET_LOCAL 1, OP_ADD, OP_PRINT, OP_POP, OP_POP, OP_RETURN",
		},
		{
			name:     "sombreado de global",
			source:   "var a = \"global\"; { var a = \"local\"; print a; } print a;",
			expected: "OP_CONSTANT global, OP_DEFINE_GLOBAL a, OP_CONSTANT local, OP_GET_LOCAL 0, OP_PRINT, OP_POP, OP_GET_GLOBAL a, OP_PRINT, OP_RETURN",
		},
		{
			name:     "bloques anidados y sombreado local",
			source:   "{ var a = 1; { var a = 2; print a; } print a; }",
			expected: "OP_CONSTANT 1, OP_CONSTANT 2, OP_GET_LOCAL 1, OP_PRINT, OP_POP, OP_GET_LOCAL 0, OP_PRINT, OP_POP, OP_RETURN",
		},
		{
			name:     "asignacion a local",
			source:   "{ var a = 1; a = 2; print a; }",
			expected: "OP_CONSTANT 1, OP_CONSTANT 2, OP_SET_LOCAL 0, OP_POP, OP_GET_LOCAL 0, OP_PRINT, OP_POP, OP_RETURN",
		},
		{
			name:     "local sin inicializador",
			source:   "{ var a; print a; }",
			expected: "OP_CONSTANT <nil>, OP_GET_LOCAL 0, OP_PRINT, OP_POP, OP_RETURN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk := mustCompileProgram(t, tt.source)
			if got := instructions(chunk); got != tt.expected {
				t.Errorf("instrucciones = %q, esperado %q", got, tt.expected)
			}
		})
	}
}

func TestCompileProgramTopeLocales(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("var a = 1;\n{\n")
	for i := 0; i < 256; i++ {
		fmt.Fprintf(&sb, "var x%d = a;\n", i)
	}
	sb.WriteString("}\n")
	// Con 256 variables locales compila bien.
	chunk := mustCompileProgram(t, sb.String())
	if chunk == nil {
		t.Fatal("esperaba chunk no nil para 256 locales")
	}

	// Con 257 variables locales da error de compilación.
	sb.Reset()
	sb.WriteString("var a = 1;\n{\n")
	for i := 0; i < 257; i++ {
		fmt.Fprintf(&sb, "var x%d = a;\n", i)
	}
	sb.WriteString("}\n")

	_, err := compileProgramSource(t, sb.String())
	if err == nil {
		t.Fatal("esperaba error al superar 256 variables locales")
	}
	expected := "[línea 259] Error de compilación: demasiadas variables locales en un mismo bloque (el máximo es 256)."
	if err.Error() != expected {
		t.Errorf("error = %q, esperado %q", err.Error(), expected)
	}
}
