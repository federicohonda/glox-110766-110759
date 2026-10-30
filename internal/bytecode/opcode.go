package bytecode

import "fmt"

// OpCode es una instrucción del bytecode. Ocupa un byte dentro de Chunk.Code;
// las que llevan operando lo guardan en los bytes que siguen.
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
	// la pila, relativo al comienzo del frame de la función en curso.
	OpGetLocal
	OpSetLocal

	// Variables locales capturadas por una función anidada: el slot guarda
	// una *Cell en vez del valor. OpBox envuelve en una celda nueva el valor
	// que está en el slot; OpGetCell y OpSetCell leen y escriben a través de
	// ella. Operando: el slot.
	OpBox
	OpGetCell
	OpSetCell

	// Variables capturadas vistas desde la función interna. Operando: el
	// índice en Closure.Captured de la closure en curso.
	OpGetCaptured
	OpSetCaptured

	// OpClosure apila una closure nueva. Operandos: el índice de la Function
	// en Chunk.Constants y, por cada variable que captura, un par de bytes
	// (1 = un slot del frame actual / 0 = un Captured de la closure actual,
	// índice).
	OpClosure

	// OpCall invoca al valor que está debajo de los argumentos. Operando: la
	// cantidad de argumentos.
	OpCall

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

	// Bucles: lleva como operando un offset de 2 bytes (uint16) hacia atrás.
	OpLoop

	// OpReturn desapila el valor de retorno, descarta el frame de la función
	// en curso y deja el valor donde estaba el callee. En el chunk del
	// programa termina la ejecución.
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
	OpBox:          "OP_BOX",
	OpGetCell:      "OP_GET_CELL",
	OpSetCell:      "OP_SET_CELL",
	OpGetCaptured:  "OP_GET_CAPTURED",
	OpSetCaptured:  "OP_SET_CAPTURED",
	OpClosure:      "OP_CLOSURE",
	OpCall:         "OP_CALL",
	OpNot:          "OP_NOT",
	OpEqual:        "OP_EQUAL",
	OpGreater:      "OP_GREATER",
	OpGreaterEqual: "OP_GREATER_EQUAL",
	OpLess:         "OP_LESS",
	OpLessEqual:    "OP_LESS_EQUAL",
	OpJump:         "OP_JUMP",
	OpJumpIfFalse:  "OP_JUMP_IF_FALSE",
	OpJumpIfTrue:   "OP_JUMP_IF_TRUE",
	OpLoop:         "OP_LOOP",
	OpReturn:       "OP_RETURN",
}

func (op OpCode) String() string {
	if int(op) >= len(opCodeNames) {
		return fmt.Sprintf("OpCode(%d)", byte(op))
	}
	return opCodeNames[op]
}
