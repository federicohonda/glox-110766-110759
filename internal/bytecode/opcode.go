package bytecode

import "fmt"

// OpCode es una instrucción del bytecode. Ocupa un byte dentro de Chunk.Code;
// las que llevan operando lo guardan en los bytes que siguen (por ahora solo
// OpConstant, con un operando de un byte).
type OpCode byte

const (
	// OpConstant apila una constante. Operando: el índice de la constante en
	// Chunk.Constants.
	OpConstant OpCode = iota

	// Las aritméticas binarias desapilan el operando derecho, después el
	// izquierdo, y apilan el resultado.
	OpAdd
	OpSubtract
	OpMultiply
	OpDivide
	OpModulo

	// OpNegate reemplaza el tope de la pila por su opuesto.
	OpNegate

	// OpReturn termina la ejecución del chunk.
	OpReturn
)

var opCodeNames = [...]string{
	OpConstant: "OP_CONSTANT",
	OpAdd:      "OP_ADD",
	OpSubtract: "OP_SUBTRACT",
	OpMultiply: "OP_MULTIPLY",
	OpDivide:   "OP_DIVIDE",
	OpModulo:   "OP_MODULO",
	OpNegate:   "OP_NEGATE",
	OpReturn:   "OP_RETURN",
}

func (op OpCode) String() string {
	if int(op) >= len(opCodeNames) {
		return fmt.Sprintf("OpCode(%d)", byte(op))
	}
	return opCodeNames[op]
}
