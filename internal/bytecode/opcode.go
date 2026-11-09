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

	// Clases y POO:
	// OpClass crea una nueva Class y la apila. Operando: índice constante del nombre.
	OpClass
	// OpMethod agrega el método (tope de pila) a la clase (debajo del método).
	// Operando: índice constante del nombre.
	OpMethod
	// OpGetProperty desapila una instancia y busca el campo o método.
	// Operando: índice constante del nombre.
	OpGetProperty
	// OpSetProperty asigna un campo a una instancia. Pila: [receiver, value].
	// Operando: índice constante del nombre.
	OpSetProperty
	// OpInvoke optimiza la invocación de método combinando búsqueda y llamada.
	// Operandos: índice constante del nombre (1 byte) y argc (1 byte).
	OpInvoke

	// Herencia:
	// OpInherit copia los métodos de la superclase (debajo) a la subclase
	// (tope), que se desapila. La superclase queda en la pila: es la local
	// escondida `super`.
	OpInherit
	// OpGetSuper desapila la superclase y `this`, y apila el método de la
	// superclase ligado a `this`. Operando: índice constante del nombre.
	OpGetSuper
	// OpSuperInvoke es `super.m(args)` en una sola instrucción. Pila:
	// [this, args..., superclase]. Operandos: índice constante del nombre y argc.
	OpSuperInvoke

	// Listas (la funcionalidad extra):
	// OpList desapila n elementos y apila una lista nueva con ellos, en orden.
	// Operando: n (un byte).
	OpList
	// OpGetIndex desapila el índice y la lista, y apila el elemento.
	OpGetIndex
	// OpSetIndex desapila el valor, el índice y la lista, asigna el elemento
	// y deja el valor (la asignación es una expresión).
	OpSetIndex
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
	OpClass:        "OP_CLASS",
	OpMethod:       "OP_METHOD",
	OpGetProperty:  "OP_GET_PROPERTY",
	OpSetProperty:  "OP_SET_PROPERTY",
	OpInvoke:       "OP_INVOKE",
	OpInherit:      "OP_INHERIT",
	OpGetSuper:     "OP_GET_SUPER",
	OpSuperInvoke:  "OP_SUPER_INVOKE",
	OpList:         "OP_LIST",
	OpGetIndex:     "OP_GET_INDEX",
	OpSetIndex:     "OP_SET_INDEX",
}

func (op OpCode) String() string {
	if int(op) >= len(opCodeNames) {
		return fmt.Sprintf("OpCode(%d)", byte(op))
	}
	return opCodeNames[op]
}
