package bytecode_test

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/federicohonda/glox-110766-110759/internal/bytecode"
)

// runSource compila una expresión y la ejecuta en una VM nueva.
func runSource(t *testing.T, source string) (bytecode.Value, error) {
	t.Helper()
	return bytecode.NewVM().Run(mustCompile(t, source))
}

func mustRun(t *testing.T, source string) bytecode.Value {
	t.Helper()
	value, err := runSource(t, source)
	if err != nil {
		t.Fatalf("Run(%q) falló con error: %v", source, err)
	}
	return value
}

func TestVMAritmetica(t *testing.T) {
	tests := []struct {
		source   string
		expected bytecode.Value
	}{
		{"7", 7.0},
		{"1 + 2", 3.0},
		{"5 - 8", -3.0},
		{"6 * 7", 42.0},
		{"9 / 2", 4.5},
		{"10 % 4", 2.0},
		{"-5", -5.0},
		{"--5", 5.0},
		{"1.5 + 2.25", 3.75},

		// El checkpoint de la parte: la misma expresión que en el compilador.
		{"2 + 3 * 4", 14.0},
		{"(2 + 3) * 4", 20.0},

		// El orden de los operandos importa en las no conmutativas: el que se
		// desapila primero es el derecho.
		{"1 - 2 - 3", -4.0},
		{"8 / 4 / 2", 1.0},
		{"7 % 4 % 2", 1.0},
		{"-(1 + 2)", -3.0},
		{"-2 * 3", -6.0},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := mustRun(t, tt.source); got != tt.expected {
				t.Errorf("Run(%q) = %v (%T), esperado %v (%T)", tt.source, got, got, tt.expected, tt.expected)
			}
		})
	}
}

func TestVMValoresNoNumericos(t *testing.T) {
	tests := []struct {
		source   string
		expected bytecode.Value
	}{
		{`"hola"`, "hola"},
		{`"ho" + "la"`, "hola"},
		{`"a" + "b" + "c"`, "abc"},
		{"true", true},
		{"false", false},
		{"nil", nil},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := mustRun(t, tt.source); got != tt.expected {
				t.Errorf("Run(%q) = %v (%T), esperado %v (%T)", tt.source, got, got, tt.expected, tt.expected)
			}
		})
	}
}

// Los mensajes son los mismos que da el intérprete tree-walk para la misma
// expresión.
func TestVMErroresDeRuntime(t *testing.T) {
	tests := []struct {
		source   string
		expected string
	}{
		{`1 + "a"`, "[línea 1] Error en tiempo de ejecución en '+': los operandos de '+' deben ser dos números o dos cadenas, se obtuvo: 1 y a"},
		{"nil + nil", "[línea 1] Error en tiempo de ejecución en '+': los operandos de '+' deben ser dos números o dos cadenas, se obtuvo: nil y nil"},
		{`"a" - 1`, "[línea 1] Error en tiempo de ejecución en '-': los operandos de '-' deben ser números, se obtuvo: a y 1"},
		{"2 * true", "[línea 1] Error en tiempo de ejecución en '*': los operandos de '*' deben ser números, se obtuvo: 2 y true"},
		{`-"a"`, "[línea 1] Error en tiempo de ejecución en '-': el operando de '-' debe ser un número, se obtuvo: a"},
		{"1 / 0", "[línea 1] Error en tiempo de ejecución en '/': división por cero."},
		{"1 % 0", "[línea 1] Error en tiempo de ejecución en '%': módulo por cero."},
		// La línea es la del operador que falló, no la del final de la expresión.
		{"1 +\n2 /\n0", "[línea 2] Error en tiempo de ejecución en '/': división por cero."},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			value, err := runSource(t, tt.source)
			if value != nil {
				t.Errorf("se esperaba un valor nil junto con el error, se obtuvo %v", value)
			}
			var runtimeErr *bytecode.RuntimeError
			if !errors.As(err, &runtimeErr) {
				t.Fatalf("se esperaba un *RuntimeError, se obtuvo: %v", err)
			}
			if err.Error() != tt.expected {
				t.Errorf("mensaje = %q\n esperado %q", err.Error(), tt.expected)
			}
		})
	}
}

// Una misma VM puede ejecutar varios chunks seguidos (es lo que va a hacer el
// REPL): el resultado de un Run no depende de lo que haya dejado el anterior
// en la pila, tampoco si terminó en error con valores a medio calcular.
func TestVMSeReutilizaEntreChunks(t *testing.T) {
	vm := bytecode.NewVM()

	if got, err := vm.Run(mustCompile(t, "1 + 2")); err != nil || got != 3.0 {
		t.Fatalf("primer Run = %v, %v; esperado 3", got, err)
	}
	if _, err := vm.Run(mustCompile(t, "1 + 2 + (3 / 0)")); err == nil {
		t.Fatal("el segundo Run tenía que fallar por división por cero")
	}
	if got, err := vm.Run(mustCompile(t, "10 * 4")); err != nil || got != 40.0 {
		t.Fatalf("tercer Run = %v, %v; esperado 40", got, err)
	}
}

// La VM se puede ejecutar sobre un chunk armado a mano, sin pasar por el
// compilador.
func TestVMChunkArmadoAMano(t *testing.T) {
	var chunk bytecode.Chunk
	writeConstant(&chunk, 1.2, 1)
	writeConstant(&chunk, 3.4, 1)
	chunk.WriteOp(bytecode.OpAdd, 1)
	chunk.WriteOp(bytecode.OpNegate, 1)
	chunk.WriteOp(bytecode.OpReturn, 1)

	got, err := bytecode.NewVM().Run(&chunk)
	if err != nil {
		t.Fatalf("Run falló con error: %v", err)
	}
	if got != -(1.2 + 3.4) {
		t.Errorf("Run = %v, esperado %v", got, -(1.2 + 3.4))
	}
}

func TestVMChunkMalFormado(t *testing.T) {
	t.Run("opcode desconocido", func(t *testing.T) {
		var chunk bytecode.Chunk
		chunk.Write(250, 1)

		_, err := bytecode.NewVM().Run(&chunk)
		expected := "la VM encontró un opcode desconocido (250) en el offset 0"
		if err == nil || err.Error() != expected {
			t.Errorf("error = %v, esperado %q", err, expected)
		}
	})

	t.Run("sin OpReturn", func(t *testing.T) {
		var chunk bytecode.Chunk
		writeConstant(&chunk, 1.0, 1)

		_, err := bytecode.NewVM().Run(&chunk)
		expected := "el chunk terminó sin un OP_RETURN"
		if err == nil || err.Error() != expected {
			t.Errorf("error = %v, esperado %q", err, expected)
		}
	})

	t.Run("vacío", func(t *testing.T) {
		if _, err := bytecode.NewVM().Run(&bytecode.Chunk{}); err == nil {
			t.Error("un chunk vacío tenía que dar error")
		}
	})
}

func TestVMParentesisAnidados(t *testing.T) {
	tests := []struct {
		source   string
		expected float64
	}{
		{"((1 + 2) * (3 - (8 / (6 % 4))))", -3},
		{"1 + (2 + (3 + (4 + 5)))", 15},
		{"(((1 + 2) + 3) + 4) + 5", 15},
		{"2 * (3 + 4) * 5", 70},
		{"2 * 3 + 4 * 5", 26},
		{"-(-(-(1)))", -1},
		{"-((1 + 2) * -(3 - 4))", -3},
		{"(1 + 2) % (3 * 4)", 3},
		{"(2 + 3) * (4 + 5) - (6 / (1 + 1))", 42},
		{"((((((7))))))", 7},
		// Los mismos números con otra agrupación dan otra cosa.
		{"100 / 10 / 5", 2},
		{"100 / (10 / 5)", 50},
		{"10 - 4 - 3", 3},
		{"10 - (4 - 3)", 9},
		{strings.Repeat("(", 500) + "1 + 2 * 3" + strings.Repeat(")", 500), 7},
	}

	for _, tt := range tests {
		name := tt.source
		if len(name) > 40 {
			name = name[:40]
		}
		t.Run(name, func(t *testing.T) {
			if got := mustRun(t, tt.source); got != tt.expected {
				t.Errorf("Run(%q) = %v, esperado %v", tt.source, got, tt.expected)
			}
		})
	}
}

// En glox dividir por cero es un error de runtime (en el Lox del libro da
// infinito). Vale para cualquier forma de llegar a un divisor cero.
func TestVMDivisionYModuloPorCero(t *testing.T) {
	tests := []struct {
		source   string
		expected string
	}{
		{"1 / 0", "[línea 1] Error en tiempo de ejecución en '/': división por cero."},
		{"0 / 0", "[línea 1] Error en tiempo de ejecución en '/': división por cero."},
		{"1 / (2 - 2)", "[línea 1] Error en tiempo de ejecución en '/': división por cero."},
		{"1 / (0 * 5)", "[línea 1] Error en tiempo de ejecución en '/': división por cero."},
		// El cero negativo también es cero.
		{"1 / -0", "[línea 1] Error en tiempo de ejecución en '/': división por cero."},
		{"5 % 0", "[línea 1] Error en tiempo de ejecución en '%': módulo por cero."},
		{"0 % 0", "[línea 1] Error en tiempo de ejecución en '%': módulo por cero."},
		{"5 % (3 - 3)", "[línea 1] Error en tiempo de ejecución en '%': módulo por cero."},
		// Lo que ya se había calculado antes de la división no la salva.
		{"1 + 2 * 3 - 4 / 0", "[línea 1] Error en tiempo de ejecución en '/': división por cero."},
		{"8 / 4 /\n0", "[línea 1] Error en tiempo de ejecución en '/': división por cero."},
		{"8 / 4\n/ 0", "[línea 2] Error en tiempo de ejecución en '/': división por cero."},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			_, err := runSource(t, tt.source)
			var runtimeErr *bytecode.RuntimeError
			if !errors.As(err, &runtimeErr) {
				t.Fatalf("se esperaba un *RuntimeError, se obtuvo: %v", err)
			}
			if err.Error() != tt.expected {
				t.Errorf("mensaje = %q\n esperado %q", err.Error(), tt.expected)
			}
		})
	}
}

// Con el cero como dividendo no hay ningún error.
func TestVMCeroComoDividendo(t *testing.T) {
	for _, source := range []string{"0 / 5", "0 % 5", "0 / -5", "(1 - 1) / 2"} {
		t.Run(source, func(t *testing.T) {
			if got := mustRun(t, source); got != 0.0 {
				t.Errorf("Run(%q) = %v, esperado 0", source, got)
			}
		})
	}
}

// El módulo toma el signo del dividendo y acepta decimales (math.Mod).
func TestVMModulo(t *testing.T) {
	tests := []struct {
		source   string
		expected float64
	}{
		{"5 % 3", 2},
		{"-5 % 3", -2},
		{"5 % -3", 2},
		{"-5 % -3", -2},
		{"5.5 % 2", 1.5},
		{"3 % 5", 3},
		{"6 % 3", 0},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := mustRun(t, tt.source); got != tt.expected {
				t.Errorf("Run(%q) = %v, esperado %v", tt.source, got, tt.expected)
			}
		})
	}
}

func TestVMErroresDeTipoEnCadaOperador(t *testing.T) {
	const prefix = "[línea 1] Error en tiempo de ejecución en "
	tests := []struct {
		source   string
		expected string
	}{
		{`"a" + 1`, "'+': los operandos de '+' deben ser dos números o dos cadenas, se obtuvo: a y 1"},
		{"true + false", "'+': los operandos de '+' deben ser dos números o dos cadenas, se obtuvo: true y false"},
		{`"a" + nil`, "'+': los operandos de '+' deben ser dos números o dos cadenas, se obtuvo: a y nil"},
		{`1 - "a"`, "'-': los operandos de '-' deben ser números, se obtuvo: 1 y a"},
		{`"a" * "b"`, "'*': los operandos de '*' deben ser números, se obtuvo: a y b"},
		{`"a" / 2`, "'/': los operandos de '/' deben ser números, se obtuvo: a y 2"},
		{"2 / nil", "'/': los operandos de '/' deben ser números, se obtuvo: 2 y nil"},
		{`"a" % 2`, "'%': los operandos de '%' deben ser números, se obtuvo: a y 2"},
		{"2 % true", "'%': los operandos de '%' deben ser números, se obtuvo: 2 y true"},
		{"-nil", "'-': el operando de '-' debe ser un número, se obtuvo: nil"},
		{"-true", "'-': el operando de '-' debe ser un número, se obtuvo: true"},
		{`--"a"`, "'-': el operando de '-' debe ser un número, se obtuvo: a"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			_, err := runSource(t, tt.source)
			if err == nil {
				t.Fatalf("Run(%q) no devolvió error", tt.source)
			}
			if err.Error() != prefix+tt.expected {
				t.Errorf("mensaje = %q\n esperado %q", err.Error(), prefix+tt.expected)
			}
		})
	}
}

// Regla de oro: cuando una expresión tiene más de un error posible, se
// reporta el que aparece primero al evaluar de izquierda a derecha y de
// adentro hacia afuera.
func TestVMReglaDeOro(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{"el operando se evalúa antes de mirar los tipos", `"a" * (1 / 0)`, "división por cero."},
		{"el izquierdo antes que el derecho", `(1 / 0) + ("a" - 1)`, "división por cero."},
		{"el izquierdo antes que el derecho, al revés", `("a" - 1) + (1 / 0)`, "los operandos de '-' deben ser números, se obtuvo: a y 1"},
		{"los tipos antes que el divisor", `"a" / 0`, "los operandos de '/' deben ser números, se obtuvo: a y 0"},
		{"el unario antes que el binario que lo contiene", `-"a" + (1 / 0)`, "el operando de '-' debe ser un número, se obtuvo: a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runSource(t, tt.source)
			if err == nil {
				t.Fatalf("Run(%q) no devolvió error", tt.source)
			}
			if !strings.HasSuffix(err.Error(), ": "+tt.expected) {
				t.Errorf("Run(%q) dio %q, se esperaba que termine en %q", tt.source, err.Error(), tt.expected)
			}
		})
	}
}

// Los números son float64: la aritmética tiene sus mismas rarezas, y la VM no
// las tiene que esconder ni convertir en errores.
func TestVMPuntoFlotante(t *testing.T) {
	t.Run("0.1 + 0.2 no es exactamente 0.3", func(t *testing.T) {
		a, b := 0.1, 0.2
		if got := mustRun(t, "0.1 + 0.2"); got != a+b || got == 0.3 {
			t.Errorf("Run = %v, esperado %v", got, a+b)
		}
	})

	t.Run("cero negativo", func(t *testing.T) {
		for _, source := range []string{"-0", "0 * -1", "-(1 - 1)"} {
			got, ok := mustRun(t, source).(float64)
			if !ok || got != 0 || !math.Signbit(got) {
				t.Errorf("Run(%q) = %v, esperado -0", source, got)
			}
		}
	})

	huge := "1" + strings.Repeat("0", 200)

	t.Run("irse de rango da infinito, no un error", func(t *testing.T) {
		if got, ok := mustRun(t, huge+" * "+huge).(float64); !ok || !math.IsInf(got, 1) {
			t.Errorf("Run = %v, esperado +Inf", got)
		}
		if got, ok := mustRun(t, "-"+huge+" * "+huge).(float64); !ok || !math.IsInf(got, -1) {
			t.Errorf("Run = %v, esperado -Inf", got)
		}
	})

	t.Run("infinito menos infinito da NaN", func(t *testing.T) {
		source := huge + " * " + huge + " - " + huge + " * " + huge
		if got, ok := mustRun(t, source).(float64); !ok || !math.IsNaN(got) {
			t.Errorf("Run = %v, esperado NaN", got)
		}
	})
}

func TestVMStrings(t *testing.T) {
	tests := []struct {
		source   string
		expected string
	}{
		{`""`, ""},
		{`"" + ""`, ""},
		{`"a" + ""`, "a"},
		// Sin coerción: concatenar dos strings con dígitos no los suma.
		{`"1" + "2"`, "12"},
		{`("a" + "b") + ("c" + "d")`, "abcd"},
		{"\"dos\nlíneas\" + \"!\"", "dos\nlíneas!"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := mustRun(t, tt.source); got != tt.expected {
				t.Errorf("Run(%q) = %q, esperado %q", tt.source, got, tt.expected)
			}
		})
	}
}

// Con la tabla de constantes llena (256), la VM lee bien hasta la última:
// 1 + 2 + ... + 256.
func TestVMConLaTablaDeConstantesLlena(t *testing.T) {
	terms := make([]string, 256)
	for i := range terms {
		terms[i] = strconv.Itoa(i + 1)
	}

	if got := mustRun(t, strings.Join(terms, " + ")); got != 32896.0 {
		t.Errorf("Run = %v, esperado 32896", got)
	}
}

// La pila arranca con lugar para 256 valores pero no es un tope. El
// compilador no puede generar una pila más profunda que eso (cada valor
// apilado gasta una constante), así que se arma el chunk a mano: se apila
// mil veces la misma constante y después se suma todo.
func TestVMLaPilaCreceMasAllaDeLaCapacidadInicial(t *testing.T) {
	const depth = 1000

	var chunk bytecode.Chunk
	index := byte(chunk.AddConstant(1.0))
	for range depth {
		chunk.WriteOp(bytecode.OpConstant, 1)
		chunk.Write(index, 1)
	}
	for range depth - 1 {
		chunk.WriteOp(bytecode.OpAdd, 1)
	}
	chunk.WriteOp(bytecode.OpReturn, 1)

	got, err := bytecode.NewVM().Run(&chunk)
	if err != nil {
		t.Fatalf("Run falló con error: %v", err)
	}
	if got != float64(depth) {
		t.Errorf("Run = %v, esperado %d", got, depth)
	}
}

func runProgram(t *testing.T, source string) (string, error) {
	t.Helper()
	var buf strings.Builder
	vm := bytecode.NewVM(&buf)
	chunk := mustCompileProgram(t, source)
	_, err := vm.Run(chunk)
	return buf.String(), err
}

func mustRunProgram(t *testing.T, source string) string {
	t.Helper()
	out, err := runProgram(t, source)
	if err != nil {
		t.Fatalf("RunProgram(%q) falló con error: %v", source, err)
	}
	return out
}

func TestVMGlobalesYPrint(t *testing.T) {
	tests := []struct {
		source   string
		expected string
	}{
		{"print 1;", "1\n"},
		{"var x = 1; print x;", "1\n"},
		{"var x = \"hola\"; print x; x = \"mundo\"; print x;", "hola\nmundo\n"},
		{"var x; print x;", "nil\n"},
		{"var a = 1; var b = 2; print a + b;", "3\n"},
		{"var a; var b; print a = b = 3; print a; print b;", "3\n3\n3\n"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := mustRunProgram(t, tt.source); got != tt.expected {
				t.Errorf("RunProgram(%q) = %q, esperado %q", tt.source, got, tt.expected)
			}
		})
	}
}

func TestVMGlobalNoDefinida(t *testing.T) {
	t.Run("lectura", func(t *testing.T) {
		_, err := runProgram(t, "print x;")
		if err == nil {
			t.Fatal("se esperaba error de variable no definida")
		}
		expected := "[línea 1] Error en tiempo de ejecución en 'x': variable no definida 'x'."
		if err.Error() != expected {
			t.Errorf("error = %q, esperado %q", err.Error(), expected)
		}
	})

	t.Run("asignación", func(t *testing.T) {
		_, err := runProgram(t, "x = 5;")
		if err == nil {
			t.Fatal("se esperaba error de variable no definida")
		}
		expected := "[línea 1] Error en tiempo de ejecución en 'x': variable no definida 'x'."
		if err.Error() != expected {
			t.Errorf("error = %q, esperado %q", err.Error(), expected)
		}
	})
}

func TestVMGlobalSobreviveEntreCorridas(t *testing.T) {
	var buf strings.Builder
	vm := bytecode.NewVM(&buf)

	chunk1 := mustCompileProgram(t, "var a = 10;")
	if _, err := vm.Run(chunk1); err != nil {
		t.Fatalf("primera corrida falló: %v", err)
	}

	chunk2 := mustCompileProgram(t, "print a + 5;")
	if _, err := vm.Run(chunk2); err != nil {
		t.Fatalf("segunda corrida falló: %v", err)
	}

	if got := buf.String(); got != "15\n" {
		t.Errorf("salida = %q, esperado %q", got, "15\n")
	}
}

func TestVMRedeclaracionGlobal(t *testing.T) {
	out := mustRunProgram(t, "var a = 1; var a = 2; print a;")
	if out != "2\n" {
		t.Errorf("salida = %q, esperado %q", out, "2\n")
	}
}

// TestVMErrorLineaEnInstruccionConOperando verifica que un error en una
// instrucción con operando (OpGetGlobal) reporte la línea exacta del fuente,
// asegurando que se tome de Lines[offset] y no de ip - 1.
func TestVMErrorLineaEnInstruccionConOperando(t *testing.T) {
	source := "\n\n\nprint x;"
	_, err := runProgram(t, source)
	if err == nil {
		t.Fatal("se esperaba error de variable no definida")
	}
	expected := "[línea 4] Error en tiempo de ejecución en 'x': variable no definida 'x'."
	if err.Error() != expected {
		t.Errorf("error = %q, esperado %q", err.Error(), expected)
	}
}

func TestVMLocales(t *testing.T) {
	t.Run("sombreado fase 3", func(t *testing.T) {
		source := `
var a = "global a";
var b = "global b";
var c = "global c";
{
  var a = "outer a";
  var b = "outer b";
  {
    var a = "inner a";
    print a;
    print b;
    print c;
  }
  print a;
  print b;
  print c;
}
print a;
print b;
print c;
`
		expected := "inner a\nouter b\nglobal c\nouter a\nouter b\nglobal c\nglobal a\nglobal b\nglobal c\n"
		got := mustRunProgram(t, source)
		if got != expected {
			t.Errorf("salida = %q, esperado %q", got, expected)
		}
	})

	t.Run("asignacion y scopes anidados", func(t *testing.T) {
		source := `
var x = 10;
{
  var x = 20;
  x = x + 5;
  print x;
}
print x;
`
		expected := "25\n10\n"
		got := mustRunProgram(t, source)
		if got != expected {
			t.Errorf("salida = %q, esperado %q", got, expected)
		}
	})

	t.Run("scopes hermanos", func(t *testing.T) {
		source := `
var a = "global";
{
  var b = "local1";
  print b;
}
{
  var c = "local2";
  print c;
  print a;
}
`
		expected := "local1\nlocal2\nglobal\n"
		got := mustRunProgram(t, source)
		if got != expected {
			t.Errorf("salida = %q, esperado %q", got, expected)
		}
	})
}

// TestVMReutilizacionConLocales verifica que al reutilizar una misma VM entre
// dos corridas que usan variables locales, la pila se reinicie a cero al comenzar
// la segunda corrida. Si no se reiniciara, la segunda corrida buscaría las locales
// en slots desfasados.
func TestVMReutilizacionConLocales(t *testing.T) {
	var buf strings.Builder
	vm := bytecode.NewVM(&buf)

	chunk1 := mustCompileProgram(t, "{ var a = 1; var b = 2; print a + b; }")
	if _, err := vm.Run(chunk1); err != nil {
		t.Fatalf("primera corrida falló: %v", err)
	}

	chunk2 := mustCompileProgram(t, "{ var x = 10; print x; }")
	if _, err := vm.Run(chunk2); err != nil {
		t.Fatalf("segunda corrida falló: %v", err)
	}

	expected := "3\n10\n"
	if got := buf.String(); got != expected {
		t.Errorf("salida = %q, esperado %q", got, expected)
	}
}

func TestVMComparaciones(t *testing.T) {
	tests := []struct {
		source   string
		expected string
	}{
		{"print !true;", "false\n"},
		{"print !false;", "true\n"},
		{"print !nil;", "true\n"},
		{"print !0;", "false\n"},
		{"print 1 < 2;", "true\n"},
		{"print 2 < 1;", "false\n"},
		{"print 1 <= 1;", "true\n"},
		{"print 2 <= 1;", "false\n"},
		{"print 2 > 1;", "true\n"},
		{"print 1 > 2;", "false\n"},
		{"print 1 >= 1;", "true\n"},
		{"print 1 >= 2;", "false\n"},
		{"print 1 == 1;", "true\n"},
		{"print 1 == 2;", "false\n"},
		{"print 1 != 2;", "true\n"},
		{"print 1 != 1;", "false\n"},
		{`print "a" == "a";`, "true\n"},
		{`print "a" == "b";`, "false\n"},
		{"print nil == nil;", "true\n"},
		{"print nil == false;", "false\n"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := mustRunProgram(t, tt.source); got != tt.expected {
				t.Errorf("RunProgram(%q) = %q, esperado %q", tt.source, got, tt.expected)
			}
		})
	}

	t.Run("error de tipos en comparacion", func(t *testing.T) {
		_, err := runProgram(t, `print 1 < "dos";`)
		if err == nil {
			t.Fatal("se esperaba error de tipos")
		}
		expected := `[línea 1] Error en tiempo de ejecución en '<': los operandos de '<' deben ser números, se obtuvo: 1 y dos`
		if err.Error() != expected {
			t.Errorf("error = %q, esperado %q", err.Error(), expected)
		}
	})
}

func TestVMIf(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{"if verdadero", "if (true) print 1;", "1\n"},
		{"if falso", "if (false) print 1;", ""},
		{"if else rama then", "if (1 < 2) print 1; else print 2;", "1\n"},
		{"if else rama else", "if (1 > 2) print 1; else print 2;", "2\n"},
		{
			"if con bloques y variables locales",
			"if (10 > 5) { var a = 1; print a; } else { var a = 2; print a; }",
			"1\n",
		},
		{
			"if anidado",
			`if (true) if (false) print "no"; else print "si";`,
			"si\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mustRunProgram(t, tt.source); got != tt.expected {
				t.Errorf("RunProgram(%q) = %q, esperado %q", tt.source, got, tt.expected)
			}
		})
	}
}

func TestVMLogicosCortocircuito(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{"and con cortocircuito no evalua divisor cero", "print false and (1 / 0);", "false\n"},
		{"or con cortocircuito no evalua divisor cero", "print true or (1 / 0);", "true\n"},
		{"and devuelve primer operando falso", "print nil and 1;", "nil\n"},
		{"and devuelve segundo operando si el primero es verdadero", "print 1 and 2;", "2\n"},
		{"or devuelve primer operando verdadero", `print "hola" or "chau";`, "hola\n"},
		{"or devuelve segundo operando si el primero es falso", `print nil or "defecto";`, "defecto\n"},
		{"and encadenado", "print 1 and 2 and 3;", "3\n"},
		{"or encadenado", "print false or nil or 42;", "42\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mustRunProgram(t, tt.source); got != tt.expected {
				t.Errorf("RunProgram(%q) = %q, esperado %q", tt.source, got, tt.expected)
			}
		})
	}
}
