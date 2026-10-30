package bytecode

import (
	"fmt"
	"strings"
)

// Disassemble devuelve el listado legible de todo el chunk, una instrucción
// por línea, bajo un encabezado con el nombre que se le pase:
//
//	== prueba ==
//	0000    1 OP_CONSTANT         0 '1.2'
//	0002    | OP_NEGATE
//	0003    2 OP_RETURN
//
// Las columnas son: offset del opcode dentro de Code, línea del fuente (`|`
// si es la misma que la del byte anterior), nombre de la instrucción y, si
// lleva operando, el operando junto con el valor al que refiere.
//
// Después del listado van los de las funciones declaradas en el chunk (las
// Function de su tabla de constantes), cada una bajo `== <fn nombre> ==`, y
// así recursivamente con las que estén anidadas en ellas.
func (c *Chunk) Disassemble(name string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "== %s ==\n", name)
	for offset := 0; offset < len(c.Code); {
		var text string
		text, offset = c.DisassembleInstruction(offset)
		sb.WriteString(text)
		sb.WriteByte('\n')
	}
	for _, constant := range c.Constants {
		if fn, ok := constant.(*Function); ok {
			sb.WriteString(fn.Chunk.Disassemble(fn.String()))
		}
	}
	return sb.String()
}

// DisassembleInstruction devuelve el texto de la instrucción que empieza en
// offset y el offset de la instrucción siguiente. Como las instrucciones no
// tienen todas el mismo largo, ese segundo valor es la única forma de saber
// dónde empieza la próxima.
func (c *Chunk) DisassembleInstruction(offset int) (string, int) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%04d ", offset)
	if offset > 0 && c.Lines[offset] == c.Lines[offset-1] {
		sb.WriteString("   | ")
	} else {
		fmt.Fprintf(&sb, "%4d ", c.Lines[offset])
	}

	switch op := OpCode(c.Code[offset]); op {
	case OpConstant, OpDefineGlobal, OpGetGlobal, OpSetGlobal:
		return c.constantInstruction(&sb, op, offset)

	case OpGetLocal, OpSetLocal, OpBox, OpGetCell, OpSetCell, OpGetCaptured, OpSetCaptured, OpCall:
		return c.byteInstruction(&sb, op, offset)

	case OpClosure:
		return c.closureInstruction(&sb, op, offset)

	case OpJump, OpJumpIfFalse, OpJumpIfTrue:
		return c.jumpInstruction(&sb, op, offset)

	case OpLoop:
		return c.loopInstruction(&sb, op, offset)

	case OpAdd, OpSubtract, OpMultiply, OpDivide, OpModulo, OpNegate, OpPop, OpPrint, OpReturn,
		OpNot, OpEqual, OpGreater, OpGreaterEqual, OpLess, OpLessEqual:
		sb.WriteString(op.String())
		return sb.String(), offset + 1

	default:
		fmt.Fprintf(&sb, "opcode desconocido %d", c.Code[offset])
		return sb.String(), offset + 1
	}
}

// jumpInstruction formatea una instrucción de salto cuyo operando es un offset
// de 2 bytes hacia adelante.
func (c *Chunk) jumpInstruction(sb *strings.Builder, op OpCode, offset int) (string, int) {
	if offset+2 >= len(c.Code) {
		fmt.Fprintf(sb, "%-16s <falta el operando>", op)
		return sb.String(), len(c.Code)
	}
	jump := int(uint16(c.Code[offset+1])<<8 | uint16(c.Code[offset+2]))
	target := offset + 3 + jump
	fmt.Fprintf(sb, "%-16s %4d -> %d", op, offset, target)
	return sb.String(), offset + 3
}

// loopInstruction formatea una instrucción de bucle cuyo operando es un offset
// de 2 bytes hacia atrás.
func (c *Chunk) loopInstruction(sb *strings.Builder, op OpCode, offset int) (string, int) {
	if offset+2 >= len(c.Code) {
		fmt.Fprintf(sb, "%-16s <falta el operando>", op)
		return sb.String(), len(c.Code)
	}
	jump := int(uint16(c.Code[offset+1])<<8 | uint16(c.Code[offset+2]))
	target := offset + 3 - jump
	fmt.Fprintf(sb, "%-16s %4d -> %d", op, offset, target)
	return sb.String(), offset + 3
}

// closureInstruction formatea OpClosure: la función, como una constante, y
// abajo una línea por cada variable que captura, con el lugar de donde la
// saca (`local` si es un slot del frame, `captured` si es un Captured de la
// closure en curso). Esas líneas no llevan offset porque no son instrucciones.
func (c *Chunk) closureInstruction(sb *strings.Builder, op OpCode, offset int) (string, int) {
	text, next := c.constantInstruction(sb, op, offset)
	if next != offset+2 || int(c.Code[offset+1]) >= len(c.Constants) {
		return text, next
	}
	fn, ok := c.Constants[c.Code[offset+1]].(*Function)
	if !ok {
		return text, next
	}
	for i := 0; i < fn.CapturedCount; i++ {
		if next+1 >= len(c.Code) {
			sb.WriteString("\n     |                     <falta el operando>")
			return sb.String(), len(c.Code)
		}
		where := "captured"
		if c.Code[next] == 1 {
			where = "local"
		}
		fmt.Fprintf(sb, "\n     |                     %s %d", where, c.Code[next+1])
		next += 2
	}
	return sb.String(), next
}

// byteInstruction formatea una instrucción cuyo operando es un byte directo
// (por ejemplo, el índice de slot de una variable local en la pila).
func (c *Chunk) byteInstruction(sb *strings.Builder, op OpCode, offset int) (string, int) {
	if offset+1 >= len(c.Code) {
		fmt.Fprintf(sb, "%-16s <falta el operando>", op)
		return sb.String(), offset + 1
	}
	slot := int(c.Code[offset+1])
	fmt.Fprintf(sb, "%-16s %4d", op, slot)
	return sb.String(), offset + 2
}

// constantInstruction formatea una instrucción cuyo operando es un índice de
// la tabla de constantes. Un chunk mal armado (sin el byte del operando, o
// con un índice que no existe) se muestra como tal en vez de hacer explotar
// al disassembler, que es justamente la herramienta para encontrar esos
// errores.
func (c *Chunk) constantInstruction(sb *strings.Builder, op OpCode, offset int) (string, int) {
	if offset+1 >= len(c.Code) {
		fmt.Fprintf(sb, "%-16s <falta el operando>", op)
		return sb.String(), offset + 1
	}

	index := int(c.Code[offset+1])
	if index >= len(c.Constants) {
		fmt.Fprintf(sb, "%-16s %4d <constante inexistente>", op, index)
	} else {
		fmt.Fprintf(sb, "%-16s %4d '%s'", op, index, formatValue(c.Constants[index]))
	}
	return sb.String(), offset + 2
}

// formatValue muestra un valor de Lox igual que lo imprimiría `print`.
func formatValue(value Value) string {
	if value == nil {
		return "nil"
	}
	return fmt.Sprintf("%v", value)
}
