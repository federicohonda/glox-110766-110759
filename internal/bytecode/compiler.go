package bytecode

import (
	"fmt"

	"github.com/federicohonda/glox-110766-110759/internal/ast"
	"github.com/federicohonda/glox-110766-110759/internal/token"
)

// maxConstants es la cantidad de constantes que entran en un chunk: el
// operando de OpConstant es un único byte.
const maxConstants = 256

// maxLocals es la cantidad de variables locales que entran en una función
// (contando el slot del callee): los operandos de OpGetLocal y OpSetLocal son
// de un único byte.
const maxLocals = 256

// maxCaptured es la cantidad de variables que puede capturar una función: el
// operando de OpGetCaptured y OpSetCaptured es de un único byte.
const maxCaptured = 256

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
	// captured indica que alguna función anidada la usa: su slot guarda una
	// *Cell en vez del valor.
	captured bool
}

// capture dice de dónde saca OpClosure cada variable capturada: de un slot
// del frame que crea la closure (fromLocal) o del Captured de esa misma
// closure, cuando la variable viene de más afuera.
type capture struct {
	fromLocal bool
	index     int
}

// compiler recorre el AST y va escribiendo el bytecode en su chunk. Hay uno
// por función que se está compilando, encadenados por enclosing hasta el del
// programa.
type compiler struct {
	enclosing *compiler
	chunk     *Chunk
	// captures son las variables de funciones externas que usa esta función,
	// en el orden en que quedan en Closure.Captured.
	captures []capture
	// captured son las declaraciones locales que alguna función anidada usa
	// (ver findCaptured). Es el mismo mapa para todos los compiladores.
	captured map[any]bool
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
	c := &compiler{chunk: &Chunk{}, globals: make(map[string]int), captured: map[any]bool{}}
	if err := c.expression(expr); err != nil {
		return nil, err
	}
	c.emit(OpReturn, c.line)
	return c.chunk, nil
}

// CompileProgram traduce una lista de sentencias a un Chunk nuevo, terminado
// en OpReturn. Cada función declarada queda compilada a su propio chunk,
// dentro de la tabla de constantes del chunk que la declara. Se corta en el
// primer error.
func CompileProgram(stmts []ast.Stmt) (*Chunk, error) {
	c := &compiler{chunk: &Chunk{}, globals: make(map[string]int), captured: findCaptured(stmts)}
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
				return tooManyLocals(s.Name.Line)
			}
			slot := len(c.locals)
			captured := c.captured[s]
			c.locals = append(c.locals, local{name: s.Name.Lexeme, depth: c.scopeDepth, captured: captured})
			if captured {
				c.emitByte(OpBox, slot, s.Name.Line)
			}
			return nil
		}

		nameIdx, err := c.nameConstant(s.Name.Lexeme, s.Name.Line)
		if err != nil {
			return err
		}
		c.emitByte(OpDefineGlobal, nameIdx, s.Name.Line)
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
		line := s.Name.Line
		if c.scopeDepth == 0 {
			if err := c.function(s); err != nil {
				return err
			}
			nameIdx, err := c.nameConstant(s.Name.Lexeme, line)
			if err != nil {
				return err
			}
			c.emitByte(OpDefineGlobal, nameIdx, line)
			return nil
		}

		// La local se registra antes de compilar el cuerpo para que la
		// función pueda llamarse a sí misma.
		if len(c.locals) >= maxLocals {
			return tooManyLocals(line)
		}
		slot := len(c.locals)
		captured := c.captured[s]
		c.locals = append(c.locals, local{name: s.Name.Lexeme, depth: c.scopeDepth, captured: captured})
		if !captured {
			return c.function(s)
		}
		// Si se captura (a sí misma, o desde otra función anidada), la celda
		// tiene que existir antes de crear la closure: se reserva el slot con
		// nil, se envuelve, se crea la closure y recién ahí se guarda en la
		// celda.
		if err := c.constant(nil, line); err != nil {
			return err
		}
		c.emitByte(OpBox, slot, line)
		if err := c.function(s); err != nil {
			return err
		}
		c.emitByte(OpSetCell, slot, line)
		c.emit(OpPop, line)
		return nil

	case *ast.ReturnStmt:
		line := s.Keyword.Line
		if s.Value != nil {
			if err := c.expression(s.Value); err != nil {
				return err
			}
		} else if err := c.constant(nil, line); err != nil {
			return err
		}
		c.emit(OpReturn, line)
		return nil

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
		return c.variable(e.Name, false)

	case *ast.Assign:
		if err := c.expression(e.Value); err != nil {
			return err
		}
		return c.variable(e.Name, true)

	case *ast.Call:
		// Regla de oro: el callee y los argumentos se evalúan, en orden, antes
		// de chequear si el valor es invocable o si la aridad coincide.
		if err := c.expression(e.Callee); err != nil {
			return err
		}
		for _, arg := range e.Arguments {
			if err := c.expression(arg); err != nil {
				return err
			}
		}
		// El parser ya rechaza más de 255 argumentos, así que entran en el
		// byte del operando.
		c.emitByte(OpCall, len(e.Arguments), e.Paren.Line)
		return nil

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

// emitByte emite una instrucción con un operando de un byte.
func (c *compiler) emitByte(op OpCode, operand int, line int) {
	c.emit(op, line)
	c.chunk.Write(byte(operand), line)
}

// function compila la declaración a un chunk propio, con un compilador nuevo
// encadenado a c, y emite en c el OpClosure que crea la closure en runtime.
func (c *compiler) function(decl *ast.FunDecl) error {
	line := decl.Name.Line
	fn := &Function{Name: decl.Name.Lexeme, Arity: len(decl.Params), Chunk: &Chunk{}}
	fc := &compiler{
		enclosing: c,
		chunk:     fn.Chunk,
		captured:  c.captured,
		globals:   make(map[string]int),
		// El slot 0 del frame es el callee: queda reservado con un nombre que
		// ninguna variable puede tener. Los parámetros van del 1 en adelante.
		locals:     []local{{name: ""}},
		scopeDepth: 1,
		line:       line,
	}
	for i := range decl.Params {
		slot := len(fc.locals)
		captured := c.captured[&decl.Params[i]]
		fc.locals = append(fc.locals, local{name: decl.Params[i].Lexeme, depth: 1, captured: captured})
		if captured {
			// El argumento llega como valor en su slot: el prólogo lo envuelve
			// en una celda en el mismo lugar.
			fc.emitByte(OpBox, slot, decl.Params[i].Line)
		}
	}
	for _, stmt := range decl.Body {
		if err := fc.statement(stmt); err != nil {
			return err
		}
	}
	// Una función que termina sin `return` devuelve nil.
	if err := fc.constant(nil, fc.line); err != nil {
		return err
	}
	fc.emit(OpReturn, fc.line)
	fn.CapturedCount = len(fc.captures)

	if len(c.chunk.Constants) >= maxConstants {
		return &CompileError{
			Line:    line,
			Message: fmt.Sprintf("demasiadas constantes en un mismo chunk (el máximo es %d).", maxConstants),
		}
	}
	c.emitByte(OpClosure, c.chunk.AddConstant(fn), line)
	for _, cp := range fc.captures {
		fromLocal := 0
		if cp.fromLocal {
			fromLocal = 1
		}
		c.chunk.Write(byte(fromLocal), line)
		c.chunk.Write(byte(cp.index), line)
	}
	return nil
}

// variable emite la lectura (o la escritura, si set) de una variable: local
// de esta función, capturada de una función externa, o global.
func (c *compiler) variable(name token.Token, set bool) error {
	line := name.Line
	if slot := c.resolveLocal(name.Lexeme); slot != -1 {
		get, put := OpGetLocal, OpSetLocal
		if c.locals[slot].captured {
			get, put = OpGetCell, OpSetCell
		}
		if set {
			get = put
		}
		c.emitByte(get, slot, line)
		return nil
	}

	index, err := c.resolveCaptured(name.Lexeme, line)
	if err != nil {
		return err
	}
	if index != -1 {
		op := OpGetCaptured
		if set {
			op = OpSetCaptured
		}
		c.emitByte(op, index, line)
		return nil
	}

	nameIdx, err := c.nameConstant(name.Lexeme, line)
	if err != nil {
		return err
	}
	op := OpGetGlobal
	if set {
		op = OpSetGlobal
	}
	c.emitByte(op, nameIdx, line)
	return nil
}

// resolveCaptured busca name en las funciones que envuelven a esta, de
// adentro hacia afuera, y devuelve su índice en Captured (agregándola si es
// la primera vez que esta función la usa), o -1 si no es local de ninguna:
// entonces es global. Una variable de dos o más funciones más afuera se
// captura también en cada función intermedia, para que OpClosure la tenga a
// mano al crear la closure interna.
func (c *compiler) resolveCaptured(name string, line int) (int, error) {
	if c.enclosing == nil {
		return -1, nil
	}
	if slot := c.enclosing.resolveLocal(name); slot != -1 {
		if !c.enclosing.locals[slot].captured {
			return 0, fmt.Errorf("error interno del compilador: '%s' se captura pero no está en una celda", name)
		}
		return c.addCapture(true, slot, line)
	}
	index, err := c.enclosing.resolveCaptured(name, line)
	if err != nil || index == -1 {
		return index, err
	}
	return c.addCapture(false, index, line)
}

func (c *compiler) addCapture(fromLocal bool, index int, line int) (int, error) {
	for i, cp := range c.captures {
		if cp.fromLocal == fromLocal && cp.index == index {
			return i, nil
		}
	}
	if len(c.captures) >= maxCaptured {
		return 0, &CompileError{
			Line:    line,
			Message: fmt.Sprintf("demasiadas variables capturadas en una función (el máximo es %d).", maxCaptured),
		}
	}
	c.captures = append(c.captures, capture{fromLocal: fromLocal, index: index})
	return len(c.captures) - 1, nil
}

func tooManyLocals(line int) error {
	return &CompileError{
		Line:    line,
		Message: fmt.Sprintf("demasiadas variables locales en una misma función (el máximo es %d).", maxLocals),
	}
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
