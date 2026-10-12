package bytecode_test

import (
	"errors"
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

// Una misma VM puede ejecutar varios chunks seguidos (es lo que hace el REPL):
// cada Run arranca con la pila vacía, también si el anterior terminó en error.
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
