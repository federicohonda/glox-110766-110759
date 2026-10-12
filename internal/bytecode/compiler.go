package bytecode

import (
	"fmt"

	"github.com/federicohonda/glox-110766-110759/internal/ast"
	"github.com/federicohonda/glox-110766-110759/internal/token"
)

// maxConstants es la cantidad de constantes que entran en un chunk: el
// operando de OpConstant es un único byte.
const maxConstants = 256

// CompileError es un error detectado al generar bytecode: un límite del
// formato de las instrucciones, o una construcción del lenguaje que el
// compilador todavía no sabe traducir.
type CompileError struct {
	Line    int
	Message string
}

func (e *CompileError) Error() string {
	return fmt.Sprintf("[línea %d] Error de compilación: %s", e.Line, e.Message)
}

// binaryOps es la instrucción que corresponde a cada operador aritmético
// binario.
var binaryOps = map[token.TokenType]OpCode{
	token.PLUS:    OpAdd,
	token.MINUS:   OpSubtract,
	token.STAR:    OpMultiply,
	token.SLASH:   OpDivide,
	token.PERCENT: OpModulo,
}

// compiler recorre el AST y va escribiendo el bytecode en su chunk.
type compiler struct {
	chunk *Chunk
	// line es la línea de la última instrucción emitida.
	line int
}

// Compile traduce una expresión a un Chunk nuevo, terminado en OpReturn.
//
// Parte del mismo AST que usa el intérprete tree-walk: el compilador no
// vuelve a parsear, así que la precedencia y la asociatividad ya vienen
// resueltas en la forma del árbol. Se corta en el primer error.
func Compile(expr ast.Expr) (*Chunk, error) {
	c := &compiler{chunk: &Chunk{}}
	if err := c.expression(expr); err != nil {
		return nil, err
	}
	c.emit(OpReturn, c.line)
	return c.chunk, nil
}

// expression emite el código que deja el valor de la expresión en el tope de
// la pila. Es un recorrido en postorden: primero el código de los operandos y
// al final la instrucción que los consume, que es el orden en que una VM de
// pila los necesita.
func (c *compiler) expression(expr ast.Expr) error {
	switch e := expr.(type) {
	case *ast.Literal:
		return c.constant(e.Value, e.Line)

	case *ast.Grouping:
		// Los paréntesis no generan código: ya cumplieron su función al
		// darle forma al árbol.
		return c.expression(e.Expression)

	case *ast.Unary:
		if e.Operator.Type != token.MINUS {
			return unsupported(e.Operator.Line, fmt.Sprintf("el operador '%s'", e.Operator.Lexeme))
		}
		if err := c.expression(e.Right); err != nil {
			return err
		}
		c.emit(OpNegate, e.Operator.Line)
		return nil

	case *ast.Binary:
		op, ok := binaryOps[e.Operator.Type]
		if !ok {
			return unsupported(e.Operator.Line, fmt.Sprintf("el operador '%s'", e.Operator.Lexeme))
		}
		// Regla de oro de Lox: el código del operando izquierdo va antes que
		// el del derecho, así sus efectos ocurren en ese orden.
		if err := c.expression(e.Left); err != nil {
			return err
		}
		if err := c.expression(e.Right); err != nil {
			return err
		}
		c.emit(op, e.Operator.Line)
		return nil

	case *ast.Logical:
		return unsupported(e.Operator.Line, fmt.Sprintf("el operador '%s'", e.Operator.Lexeme))

	case *ast.Variable:
		return unsupported(e.Name.Line, "las variables")

	case *ast.Assign:
		return unsupported(e.Name.Line, "la asignación")

	case *ast.Call:
		return unsupported(e.Paren.Line, "las llamadas")

	default:
		return fmt.Errorf("tipo de expresión no soportado por el compilador: %T", expr)
	}
}

// constant agrega el valor a la tabla de constantes y emite el OpConstant que
// lo apila.
func (c *compiler) constant(value Value, line int) error {
	if len(c.chunk.Constants) >= maxConstants {
		return &CompileError{
			Line:    line,
			Message: fmt.Sprintf("demasiadas constantes en un mismo chunk (el máximo es %d).", maxConstants),
		}
	}
	index := c.chunk.AddConstant(value)
	c.emit(OpConstant, line)
	c.chunk.Write(byte(index), line)
	return nil
}

func (c *compiler) emit(op OpCode, line int) {
	c.chunk.WriteOp(op, line)
	c.line = line
}

// unsupported arma el error de una construcción que el lenguaje tiene pero
// que todavía no se compila a bytecode.
func unsupported(line int, what string) error {
	return &CompileError{
		Line:    line,
		Message: fmt.Sprintf("el compilador a bytecode todavía no soporta %s.", what),
	}
}
