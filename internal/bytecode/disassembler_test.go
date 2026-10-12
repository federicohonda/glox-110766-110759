package bytecode_test

import (
	"strings"
	"testing"

	"github.com/federicohonda/glox-110766-110759/internal/bytecode"
)

// writeConstant agrega la constante al chunk y emite el OpConstant que la apila.
func writeConstant(chunk *bytecode.Chunk, value bytecode.Value, line int) {
	chunk.WriteOp(bytecode.OpConstant, line)
	chunk.Write(byte(chunk.AddConstant(value)), line)
}

// Chunk armado a mano para `-(1.2 + 3.4) / 5.6`, con el divisor en otra línea.
func TestDisassembleChunkArmadoAMano(t *testing.T) {
	var chunk bytecode.Chunk
	writeConstant(&chunk, 1.2, 1)
	writeConstant(&chunk, 3.4, 1)
	chunk.WriteOp(bytecode.OpAdd, 1)
	chunk.WriteOp(bytecode.OpNegate, 1)
	writeConstant(&chunk, 5.6, 2)
	chunk.WriteOp(bytecode.OpDivide, 2)
	chunk.WriteOp(bytecode.OpReturn, 2)

	expected := strings.Join([]string{
		"== prueba ==",
		"0000    1 OP_CONSTANT         0 '1.2'",
		"0002    | OP_CONSTANT         1 '3.4'",
		"0004    | OP_ADD",
		"0005    | OP_NEGATE",
		"0006    2 OP_CONSTANT         2 '5.6'",
		"0008    | OP_DIVIDE",
		"0009    | OP_RETURN",
		"",
	}, "\n")

	if got := chunk.Disassemble("prueba"); got != expected {
		t.Errorf("Disassemble distinto\n--- obtenido\n%s--- esperado\n%s", got, expected)
	}
}

func TestDisassembleTodosLosOpcodes(t *testing.T) {
	var chunk bytecode.Chunk
	writeConstant(&chunk, 10.0, 1)
	chunk.WriteOp(bytecode.OpAdd, 2)
	chunk.WriteOp(bytecode.OpSubtract, 3)
	chunk.WriteOp(bytecode.OpMultiply, 4)
	chunk.WriteOp(bytecode.OpDivide, 5)
	chunk.WriteOp(bytecode.OpModulo, 6)
	chunk.WriteOp(bytecode.OpNegate, 7)
	chunk.WriteOp(bytecode.OpReturn, 8)

	expected := strings.Join([]string{
		"== todos ==",
		"0000    1 OP_CONSTANT         0 '10'",
		"0002    2 OP_ADD",
		"0003    3 OP_SUBTRACT",
		"0004    4 OP_MULTIPLY",
		"0005    5 OP_DIVIDE",
		"0006    6 OP_MODULO",
		"0007    7 OP_NEGATE",
		"0008    8 OP_RETURN",
		"",
	}, "\n")

	if got := chunk.Disassemble("todos"); got != expected {
		t.Errorf("Disassemble distinto\n--- obtenido\n%s--- esperado\n%s", got, expected)
	}
}

func TestDisassembleChunkVacio(t *testing.T) {
	var chunk bytecode.Chunk
	if got := chunk.Disassemble("vacío"); got != "== vacío ==\n" {
		t.Errorf("Disassemble de un chunk vacío = %q", got)
	}
}

// Las constantes se muestran como las imprimiría `print`.
func TestDisassembleFormatoDeConstantes(t *testing.T) {
	tests := []struct {
		name     string
		value    bytecode.Value
		expected string
	}{
		{"entero", 3.0, "'3'"},
		{"decimal", 0.5, "'0.5'"},
		{"negativo", -2.0, "'-2'"},
		{"string", "hola", "'hola'"},
		{"true", true, "'true'"},
		{"nil", nil, "'nil'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var chunk bytecode.Chunk
			writeConstant(&chunk, tt.value, 1)

			got, _ := chunk.DisassembleInstruction(0)
			if !strings.HasSuffix(got, tt.expected) {
				t.Errorf("DisassembleInstruction = %q, se esperaba que termine en %s", got, tt.expected)
			}
		})
	}
}

// El segundo valor de DisassembleInstruction es dónde empieza la instrucción
// siguiente: 2 bytes más adelante si lleva operando, 1 si no.
func TestDisassembleInstructionDevuelveElSiguienteOffset(t *testing.T) {
	var chunk bytecode.Chunk
	writeConstant(&chunk, 1.0, 1)
	chunk.WriteOp(bytecode.OpNegate, 1)
	chunk.WriteOp(bytecode.OpReturn, 1)

	tests := []struct {
		offset   int
		expected int
	}{
		{0, 2},
		{2, 3},
		{3, 4},
	}

	for _, tt := range tests {
		if _, next := chunk.DisassembleInstruction(tt.offset); next != tt.expected {
			t.Errorf("DisassembleInstruction(%d) siguiente = %d, esperado %d", tt.offset, next, tt.expected)
		}
	}
}

// Un chunk mal armado se tiene que poder mirar igual: es para lo que sirve el
// disassembler.
func TestDisassembleChunkMalArmado(t *testing.T) {
	t.Run("opcode desconocido", func(t *testing.T) {
		var chunk bytecode.Chunk
		chunk.Write(250, 1)
		chunk.WriteOp(bytecode.OpReturn, 1)

		expected := "== roto ==\n0000    1 opcode desconocido 250\n0001    | OP_RETURN\n"
		if got := chunk.Disassemble("roto"); got != expected {
			t.Errorf("Disassemble = %q, esperado %q", got, expected)
		}
	})

	t.Run("OpConstant sin operando", func(t *testing.T) {
		var chunk bytecode.Chunk
		chunk.WriteOp(bytecode.OpConstant, 1)

		expected := "== roto ==\n0000    1 OP_CONSTANT      <falta el operando>\n"
		if got := chunk.Disassemble("roto"); got != expected {
			t.Errorf("Disassemble = %q, esperado %q", got, expected)
		}
	})

	t.Run("índice de constante inexistente", func(t *testing.T) {
		var chunk bytecode.Chunk
		chunk.WriteOp(bytecode.OpConstant, 1)
		chunk.Write(7, 1)
		chunk.WriteOp(bytecode.OpReturn, 1)

		expected := "== roto ==\n0000    1 OP_CONSTANT         7 <constante inexistente>\n0002    | OP_RETURN\n"
		if got := chunk.Disassemble("roto"); got != expected {
			t.Errorf("Disassemble = %q, esperado %q", got, expected)
		}
	})
}
