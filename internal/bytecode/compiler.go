package bytecode

import (
	"fmt"

	"github.com/federicohonda/glox-110766-110759/internal/ast"
	"github.com/federicohonda/glox-110766-110759/internal/token"
)

// maxConstants es la cantidad de constantes que entran en un chunk: el
// operando de OpConstant es un único byte.
const maxConstants = 256

// maxLocals es la cantidad de variables locales que entran en un chunk: los
// operandos de OpGetLocal y OpSetLocal son de un único byte.
const maxLocals = 256

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
// y relacional binario (salvo BANG_EQUAL que emite OpEqual + OpNot).
var binaryOps = map[token.TokenType]OpCode{
	token.PLUS:          OpAdd,
	token.MINUS:         OpSubtract,
	token.STAR:          OpMultiply,
	token.SLASH:         OpDivide,
	token.PERCENT:       OpModulo,
	token.GREATER:       OpGreater,
	token.GREATER_EQUAL: OpGreaterEqual,
	token.LESS:          OpLess,
	token.LESS_EQUAL:    OpLessEqual,
	token.EQUAL_EQUAL:   OpEqual,
}

// local representa una variable local conocida en tiempo de compilación.
// Su posición dentro de c.locals coincide exactamente con su índice de slot
// en la pila de la VM.
type local struct {
	name  string
	depth int
}

// compiler recorre el AST y va escribiendo el bytecode en su chunk.
type compiler struct {
	chunk *Chunk
	// line es la línea de la última instrucción emitida.
	line int
	// globals deduplica los nombres de las variables globales en Constants.
	globals map[string]int
	// locals lleva la cuenta de las variables locales activas en la pila.
	locals []local
	// scopeDepth es la profundidad de anidamiento de bloques actual (0 = global).
	scopeDepth int
}

// Compile traduce una expresión a un Chunk nuevo, terminado en OpReturn.
//
// Parte del mismo AST que usa el intérprete tree-walk: el compilador no
// vuelve a parsear, así que la precedencia y la asociatividad ya vienen
// resueltas en la forma del árbol. Se corta en el primer error.
func Compile(expr ast.Expr) (*Chunk, error) {
	c := &compiler{chunk: &Chunk{}, globals: make(map[string]int)}
	if err := c.expression(expr); err != nil {
		return nil, err
	}
	c.emit(OpReturn, c.line)
	return c.chunk, nil
}

// CompileProgram traduce una lista de sentencias a un Chunk nuevo, terminado
// en OpReturn. Se corta en el primer error.
func CompileProgram(stmts []ast.Stmt) (*Chunk, error) {
	c := &compiler{chunk: &Chunk{}, globals: make(map[string]int)}
	for _, stmt := range stmts {
		if err := c.statement(stmt); err != nil {
			return nil, err
		}
	}
	line := c.line
	if line == 0 {
		line = 1
	}
	c.emit(OpReturn, line)
	return c.chunk, nil
}

// statement emite el código para una sentencia de Lox.
func (c *compiler) statement(stmt ast.Stmt) error {
	switch s := stmt.(type) {
	case *ast.PrintStmt:
		if err := c.expression(s.Expression); err != nil {
			return err
		}
		c.emit(OpPrint, c.line)
		return nil

	case *ast.ExpressionStmt:
		if err := c.expression(s.Expression); err != nil {
			return err
		}
		c.emit(OpPop, c.line)
		return nil

	case *ast.VarDecl:
		if s.Initializer != nil {
			if err := c.expression(s.Initializer); err != nil {
				return err
			}
		} else {
			if err := c.constant(nil, s.Name.Line); err != nil {
				return err
			}
		}

		if c.scopeDepth > 0 {
			if len(c.locals) >= maxLocals {
				return &CompileError{
					Line:    s.Name.Line,
					Message: fmt.Sprintf("demasiadas variables locales en un mismo bloque (el máximo es %d).", maxLocals),
				}
			}
			c.locals = append(c.locals, local{name: s.Name.Lexeme, depth: c.scopeDepth})
			return nil
		}

		nameIdx, err := c.nameConstant(s.Name.Lexeme, s.Name.Line)
		if err != nil {
			return err
		}
		c.emit(OpDefineGlobal, s.Name.Line)
		c.chunk.Write(byte(nameIdx), s.Name.Line)
		return nil

	case *ast.Block:
		c.beginScope()
		for _, nested := range s.Statements {
			if err := c.statement(nested); err != nil {
				return err
			}
		}
		c.endScope()
		return nil

	case *ast.IfStmt:
		line := exprLine(s.Condition)
		if err := c.expression(s.Condition); err != nil {
			return err
		}
		thenJump := c.emitJump(OpJumpIfFalse, line)
		c.emit(OpPop, line)

		if err := c.statement(s.ThenBranch); err != nil {
			return err
		}

		elseJump := c.emitJump(OpJump, line)
		if err := c.patchJump(thenJump, line); err != nil {
			return err
		}
		c.emit(OpPop, line)

		if s.ElseBranch != nil {
			if err := c.statement(s.ElseBranch); err != nil {
				return err
			}
		}
		if err := c.patchJump(elseJump, line); err != nil {
			return err
		}
		return nil

	case *ast.WhileStmt:
		loopStart := len(c.chunk.Code)
		line := exprLine(s.Condition)
		if err := c.expression(s.Condition); err != nil {
			return err
		}
		exitJump := c.emitJump(OpJumpIfFalse, line)
		c.emit(OpPop, line)

		if err := c.statement(s.Body); err != nil {
			return err
		}

		if err := c.emitLoop(loopStart, line); err != nil {
			return err
		}

		if err := c.patchJump(exitJump, line); err != nil {
			return err
		}
		c.emit(OpPop, line)
		return nil

	case *ast.FunDecl:
		return unsupported(s.Name.Line, "las funciones")

	case *ast.ReturnStmt:
		return unsupported(s.Keyword.Line, "la sentencia 'return'")

	default:
		return fmt.Errorf("tipo de sentencia no soportado por el compilador: %T", stmt)
	}
}

// stmtLine extrae la línea del fuente de una sentencia para reportar errores.
func stmtLine(stmt ast.Stmt) int {
	switch s := stmt.(type) {
	case *ast.PrintStmt:
		return exprLine(s.Expression)
	case *ast.ExpressionStmt:
		return exprLine(s.Expression)
	case *ast.VarDecl:
		return s.Name.Line
	case *ast.IfStmt:
		return exprLine(s.Condition)
	case *ast.WhileStmt:
		return exprLine(s.Condition)
	case *ast.Block:
		if len(s.Statements) > 0 {
			return stmtLine(s.Statements[0])
		}
	case *ast.FunDecl:
		return s.Name.Line
	case *ast.ReturnStmt:
		return s.Keyword.Line
	}
	return 1
}

// exprLine extrae la línea del fuente de una expresión para reportar errores.
func exprLine(expr ast.Expr) int {
	switch e := expr.(type) {
	case *ast.Literal:
		return e.Line
	case *ast.Grouping:
		return exprLine(e.Expression)
	case *ast.Unary:
		return e.Operator.Line
	case *ast.Binary:
		return e.Operator.Line
	case *ast.Logical:
		return e.Operator.Line
	case *ast.Variable:
		return e.Name.Line
	case *ast.Assign:
		return e.Name.Line
	case *ast.Call:
		return e.Paren.Line
	}
	return 1
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
		if err := c.expression(e.Right); err != nil {
			return err
		}
		switch e.Operator.Type {
		case token.MINUS:
			c.emit(OpNegate, e.Operator.Line)
			return nil
		case token.BANG:
			c.emit(OpNot, e.Operator.Line)
			return nil
		default:
			return unsupported(e.Operator.Line, fmt.Sprintf("el operador '%s'", e.Operator.Lexeme))
		}

	case *ast.Binary:
		// Regla de oro de Lox: el código del operando izquierdo va antes que
		// el del derecho, así sus efectos ocurren en ese orden.
		if err := c.expression(e.Left); err != nil {
			return err
		}
		if err := c.expression(e.Right); err != nil {
			return err
		}
		if e.Operator.Type == token.BANG_EQUAL {
			c.emit(OpEqual, e.Operator.Line)
			c.emit(OpNot, e.Operator.Line)
			return nil
		}
		op, ok := binaryOps[e.Operator.Type]
		if !ok {
			return unsupported(e.Operator.Line, fmt.Sprintf("el operador '%s'", e.Operator.Lexeme))
		}
		c.emit(op, e.Operator.Line)
		return nil

	case *ast.Logical:
		if err := c.expression(e.Left); err != nil {
			return err
		}
		if e.Operator.Type == token.AND {
			endJump := c.emitJump(OpJumpIfFalse, e.Operator.Line)
			c.emit(OpPop, e.Operator.Line)
			if err := c.expression(e.Right); err != nil {
				return err
			}
			return c.patchJump(endJump, e.Operator.Line)
		} else if e.Operator.Type == token.OR {
			endJump := c.emitJump(OpJumpIfTrue, e.Operator.Line)
			c.emit(OpPop, e.Operator.Line)
			if err := c.expression(e.Right); err != nil {
				return err
			}
			return c.patchJump(endJump, e.Operator.Line)
		}
		return unsupported(e.Operator.Line, fmt.Sprintf("el operador '%s'", e.Operator.Lexeme))

	case *ast.Variable:
		if slot := c.resolveLocal(e.Name.Lexeme); slot != -1 {
			c.emit(OpGetLocal, e.Name.Line)
			c.chunk.Write(byte(slot), e.Name.Line)
			return nil
		}
		nameIdx, err := c.nameConstant(e.Name.Lexeme, e.Name.Line)
		if err != nil {
			return err
		}
		c.emit(OpGetGlobal, e.Name.Line)
		c.chunk.Write(byte(nameIdx), e.Name.Line)
		return nil

	case *ast.Assign:
		if err := c.expression(e.Value); err != nil {
			return err
		}
		if slot := c.resolveLocal(e.Name.Lexeme); slot != -1 {
			c.emit(OpSetLocal, e.Name.Line)
			c.chunk.Write(byte(slot), e.Name.Line)
			return nil
		}
		nameIdx, err := c.nameConstant(e.Name.Lexeme, e.Name.Line)
		if err != nil {
			return err
		}
		c.emit(OpSetGlobal, e.Name.Line)
		c.chunk.Write(byte(nameIdx), e.Name.Line)
		return nil

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

// nameConstant busca el nombre en la tabla de constantes o lo agrega si no
// está, deduplicando los nombres de las variables globales en el chunk.
func (c *compiler) nameConstant(name string, line int) (int, error) {
	if idx, ok := c.globals[name]; ok {
		return idx, nil
	}
	if len(c.chunk.Constants) >= maxConstants {
		return 0, &CompileError{
			Line:    line,
			Message: fmt.Sprintf("demasiadas constantes en un mismo chunk (el máximo es %d).", maxConstants),
		}
	}
	idx := c.chunk.AddConstant(name)
	c.globals[name] = idx
	return idx, nil
}

func (c *compiler) emit(op OpCode, line int) {
	c.chunk.WriteOp(op, line)
	c.line = line
}

func (c *compiler) beginScope() {
	c.scopeDepth++
}

func (c *compiler) endScope() {
	c.scopeDepth--
	line := c.line
	if line == 0 {
		line = 1
	}
	for len(c.locals) > 0 && c.locals[len(c.locals)-1].depth > c.scopeDepth {
		c.emit(OpPop, line)
		c.locals = c.locals[:len(c.locals)-1]
	}
}

// resolveLocal busca una variable local por nombre recorriendo locals de
// atrás hacia adelante (desde el scope más interno al más externo). Si la
// encuentra, devuelve su índice de slot en la pila; si no, devuelve -1 para
// indicar que debe resolverse como global.
func (c *compiler) resolveLocal(name string) int {
	for i := len(c.locals) - 1; i >= 0; i-- {
		if c.locals[i].name == name {
			return i
		}
	}
	return -1
}

// emitJump escribe op seguido de 2 bytes placeholder (0xff) y devuelve el
// offset en chunk.Code del primer byte del operando para su posterior backpatching.
func (c *compiler) emitJump(op OpCode, line int) int {
	c.emit(op, line)
	c.chunk.Write(0xff, line)
	c.chunk.Write(0xff, line)
	return len(c.chunk.Code) - 2
}

// patchJump completa el operando de salto de 2 bytes en offset con la distancia
// hasta el final actual de chunk.Code.
func (c *compiler) patchJump(offset int, line int) error {
	jump := len(c.chunk.Code) - offset - 2
	if jump > 65535 {
		return &CompileError{
			Line:    line,
			Message: "cuerpo del salto demasiado grande (el máximo es 65535 bytes).",
		}
	}
	c.chunk.Code[offset] = byte((jump >> 8) & 0xff)
	c.chunk.Code[offset+1] = byte(jump & 0xff)
	return nil
}

// emitLoop escribe OpLoop seguido de 2 bytes con la distancia hacia atrás
// hasta loopStart.
func (c *compiler) emitLoop(loopStart int, line int) error {
	c.emit(OpLoop, line)
	jump := len(c.chunk.Code) - loopStart + 2
	if jump > 65535 {
		return &CompileError{
			Line:    line,
			Message: "cuerpo del bucle demasiado grande (el máximo es 65535 bytes).",
		}
	}
	c.chunk.Write(byte((jump>>8)&0xff), line)
	c.chunk.Write(byte(jump&0xff), line)
	return nil
}

// unsupported arma el error de una construcción que el lenguaje tiene pero
// que todavía no se compila a bytecode.
func unsupported(line int, what string) error {
	return &CompileError{
		Line:    line,
		Message: fmt.Sprintf("el compilador a bytecode todavía no soporta %s.", what),
	}
}
