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

	// OpPop descarta el valor del tope de la pila.
	OpPop

	// OpPrint desapila un valor y lo imprime formateado.
	OpPrint

	// Variables globales: llevan como operando el índice del nombre (string)
	// en Chunk.Constants.
	OpDefineGlobal
	OpGetGlobal
	OpSetGlobal

	// Variables locales: llevan como operando el índice del slot (un byte) en
	// la pila.
	OpGetLocal
	OpSetLocal

	// Operaciones lógicas y de comparación:
	OpNot
	OpEqual
	OpGreater
	OpGreaterEqual
	OpLess
	OpLessEqual

	// Saltos: llevan como operando un offset de 2 bytes (uint16) hacia adelante.
	OpJump
	OpJumpIfFalse
	OpJumpIfTrue

	// OpReturn termina la ejecución del chunk.
	OpReturn
)

var opCodeNames = [...]string{
	OpConstant:     "OP_CONSTANT",
	OpAdd:          "OP_ADD",
	OpSubtract:     "OP_SUBTRACT",
	OpMultiply:     "OP_MULTIPLY",
	OpDivide:       "OP_DIVIDE",
	OpModulo:       "OP_MODULO",
	OpNegate:       "OP_NEGATE",
	OpPop:          "OP_POP",
	OpPrint:        "OP_PRINT",
	OpDefineGlobal: "OP_DEFINE_GLOBAL",
	OpGetGlobal:    "OP_GET_GLOBAL",
	OpSetGlobal:    "OP_SET_GLOBAL",
	OpGetLocal:     "OP_GET_LOCAL",
	OpSetLocal:     "OP_SET_LOCAL",
	OpNot:          "OP_NOT",
	OpEqual:        "OP_EQUAL",
	OpGreater:      "OP_GREATER",
	OpGreaterEqual: "OP_GREATER_EQUAL",
	OpLess:         "OP_LESS",
	OpLessEqual:    "OP_LESS_EQUAL",
	OpJump:         "OP_JUMP",
	OpJumpIfFalse:  "OP_JUMP_IF_FALSE",
	OpJumpIfTrue:   "OP_JUMP_IF_TRUE",
	OpReturn:       "OP_RETURN",
}

func (op OpCode) String() string {
	if int(op) >= len(opCodeNames) {
		return fmt.Sprintf("OpCode(%d)", byte(op))
	}
	return opCodeNames[op]
}
