package bytecode_test

import (
	"slices"
	"testing"

	"github.com/federicohonda/glox-110766-110759/internal/bytecode"
)

func TestOpCodeString(t *testing.T) {
	tests := []struct {
		op       bytecode.OpCode
		expected string
	}{
		{bytecode.OpConstant, "OP_CONSTANT"},
		{bytecode.OpAdd, "OP_ADD"},
		{bytecode.OpSubtract, "OP_SUBTRACT"},
		{bytecode.OpMultiply, "OP_MULTIPLY"},
		{bytecode.OpDivide, "OP_DIVIDE"},
		{bytecode.OpModulo, "OP_MODULO"},
		{bytecode.OpNegate, "OP_NEGATE"},
		{bytecode.OpReturn, "OP_RETURN"},
		{bytecode.OpBox, "OP_BOX"},
		{bytecode.OpGetCell, "OP_GET_CELL"},
		{bytecode.OpSetCell, "OP_SET_CELL"},
		{bytecode.OpGetCaptured, "OP_GET_CAPTURED"},
		{bytecode.OpSetCaptured, "OP_SET_CAPTURED"},
		{bytecode.OpClosure, "OP_CLOSURE"},
		{bytecode.OpCall, "OP_CALL"},
		{bytecode.OpCode(250), "OpCode(250)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.op.String(); got != tt.expected {
				t.Errorf("OpCode(%d).String() = %q, esperado %q", byte(tt.op), got, tt.expected)
			}
		})
	}
}

func TestChunkWriteMantieneCodeYLinesParalelos(t *testing.T) {
	var chunk bytecode.Chunk
	chunk.WriteOp(bytecode.OpConstant, 1)
	chunk.Write(0, 1)
	chunk.WriteOp(bytecode.OpNegate, 2)
	chunk.WriteOp(bytecode.OpReturn, 3)

	expectedCode := []byte{byte(bytecode.OpConstant), 0, byte(bytecode.OpNegate), byte(bytecode.OpReturn)}
	if !slices.Equal(chunk.Code, expectedCode) {
		t.Errorf("Code = %v, esperado %v", chunk.Code, expectedCode)
	}

	expectedLines := []int{1, 1, 2, 3}
	if !slices.Equal(chunk.Lines, expectedLines) {
		t.Errorf("Lines = %v, esperado %v", chunk.Lines, expectedLines)
	}
}

func TestAddConstantDevuelveElIndice(t *testing.T) {
	var chunk bytecode.Chunk
	values := []bytecode.Value{1.2, "hola", true, nil}

	for expected, value := range values {
		if got := chunk.AddConstant(value); got != expected {
			t.Errorf("AddConstant(%v) = %d, esperado %d", value, got, expected)
		}
	}

	if !slices.Equal(chunk.Constants, values) {
		t.Errorf("Constants = %v, esperado %v", chunk.Constants, values)
	}
}

// Agregar dos veces el mismo valor ocupa dos lugares: la tabla no deduplica.
func TestAddConstantNoDeduplica(t *testing.T) {
	var chunk bytecode.Chunk
	first := chunk.AddConstant(7.0)
	second := chunk.AddConstant(7.0)

	if first == second {
		t.Errorf("las dos constantes quedaron en el mismo índice %d", first)
	}
	if len(chunk.Constants) != 2 {
		t.Errorf("len(Constants) = %d, esperado 2", len(chunk.Constants))
	}
}
