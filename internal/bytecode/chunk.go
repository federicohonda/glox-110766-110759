// Package bytecode contiene la representación compilada de un programa de
// Lox: el Chunk (instrucciones + constantes), su juego de instrucciones
// (OpCode) y un disassembler para inspeccionarlo.
package bytecode

// Value es cualquier valor de Lox en tiempo de ejecución. Es el mismo alias
// que usa el intérprete tree-walk (interpreter.Value), así que un valor pasa
// de un lado al otro sin conversión.
type Value = any

// Chunk es una secuencia de bytecode junto con todo lo que hace falta para
// ejecutarla y para reportar errores sobre ella.
//
// Code mezcla opcodes y operandos en un único arreglo de bytes. Las
// instrucciones no guardan valores: los literales del programa viven en
// Constants y el bytecode los referencia por índice. Lines es paralelo a
// Code: Lines[i] es la línea del fuente que generó el byte Code[i], para que
// un error de runtime pueda decir dónde ocurrió.
type Chunk struct {
	Code      []byte
	Constants []Value
	Lines     []int
}

// Write agrega un byte (un opcode o un operando) al final del chunk, anotando
// la línea del fuente de la que proviene.
func (c *Chunk) Write(b byte, line int) {
	c.Code = append(c.Code, b)
	c.Lines = append(c.Lines, line)
}

// WriteOp agrega un opcode al final del chunk.
func (c *Chunk) WriteOp(op OpCode, line int) {
	c.Write(byte(op), line)
}

// AddConstant agrega un valor a la tabla de constantes y devuelve su índice,
// que es lo que se escribe como operando de OpConstant.
func (c *Chunk) AddConstant(value Value) int {
	c.Constants = append(c.Constants, value)
	return len(c.Constants) - 1
}
