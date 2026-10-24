package bytecode

import (
	"fmt"
	"io"
	"math"
	"os"
)

// RuntimeError es un error producido al ejecutar una instrucción. Line sale
// de Chunk.Lines y Operator es el símbolo del operador en el fuente, así el
// mensaje queda idéntico al del intérprete tree-walk para el mismo programa.
type RuntimeError struct {
	Line     int
	Operator string
	Message  string
}

func (e *RuntimeError) Error() string {
	return fmt.Sprintf("[línea %d] Error en tiempo de ejecución en '%s': %s", e.Line, e.Operator, e.Message)
}

// stackInitialCapacity es el lugar reservado de entrada para la pila. No es
// un tope: si una expresión necesita más, el slice crece.
const stackInitialCapacity = 256

// VM es una máquina virtual de pila: cada instrucción toma sus operandos del
// tope de la pila y deja ahí su resultado.
type VM struct {
	chunk *Chunk
	// ip (instruction pointer) es el índice en chunk.Code del próximo byte a
	// leer.
	ip      int
	stack   []Value
	globals map[string]Value
	out     io.Writer
}

// NewVM crea una VM con la pila vacía. Si no se indica un writer, la salida
// de print va a os.Stdout.
func NewVM(out ...io.Writer) *VM {
	var w io.Writer = os.Stdout
	if len(out) > 0 && out[0] != nil {
		w = out[0]
	}
	return &VM{
		stack:   make([]Value, 0, stackInitialCapacity),
		globals: make(map[string]Value),
		out:     w,
	}
}

// Run ejecuta el chunk desde su primera instrucción y devuelve el valor que
// queda en el tope de la pila al llegar a OpReturn.
//
// La VM confía en que el chunk está bien formado, que es lo que garantiza el
// compilador: no verifica en cada instrucción que la pila tenga los operandos
// que hacen falta.
func (vm *VM) Run(chunk *Chunk) (Value, error) {
	vm.chunk = chunk
	vm.ip = 0
	vm.stack = vm.stack[:0]

	for vm.ip < len(chunk.Code) {
		// offset es dónde empieza la instrucción en curso: es el que sirve
		// para buscar su línea si falla, porque ip ya avanzó.
		offset := vm.ip
		op := OpCode(vm.readByte())

		switch op {
		case OpConstant:
			vm.push(chunk.Constants[vm.readByte()])

		case OpAdd:
			right, left := vm.pop(), vm.pop()
			switch l := left.(type) {
			case float64:
				if r, ok := right.(float64); ok {
					vm.push(l + r)
					continue
				}
			case string:
				if r, ok := right.(string); ok {
					vm.push(l + r)
					continue
				}
			}
			return nil, vm.runtimeError(offset, "+", "los operandos de '+' deben ser dos números o dos cadenas, se obtuvo: %v y %v", formatValue(left), formatValue(right))

		case OpSubtract:
			l, r, err := vm.popNumbers(offset, "-")
			if err != nil {
				return nil, err
			}
			vm.push(l - r)

		case OpMultiply:
			l, r, err := vm.popNumbers(offset, "*")
			if err != nil {
				return nil, err
			}
			vm.push(l * r)

		case OpDivide:
			l, r, err := vm.popNumbers(offset, "/")
			if err != nil {
				return nil, err
			}
			if r == 0 {
				return nil, vm.runtimeError(offset, "/", "división por cero.")
			}
			vm.push(l / r)

		case OpModulo:
			l, r, err := vm.popNumbers(offset, "%")
			if err != nil {
				return nil, err
			}
			if r == 0 {
				return nil, vm.runtimeError(offset, "%", "módulo por cero.")
			}
			vm.push(math.Mod(l, r))

		case OpNegate:
			operand := vm.pop()
			n, ok := operand.(float64)
			if !ok {
				return nil, vm.runtimeError(offset, "-", "el operando de '-' debe ser un número, se obtuvo: %v", formatValue(operand))
			}
			vm.push(-n)

		case OpPop:
			vm.pop()

		case OpPrint:
			fmt.Fprintln(vm.out, formatValue(vm.pop()))

		case OpDefineGlobal:
			name := chunk.Constants[vm.readByte()].(string)
			vm.globals[name] = vm.pop()

		case OpGetGlobal:
			name := chunk.Constants[vm.readByte()].(string)
			val, ok := vm.globals[name]
			if !ok {
				return nil, vm.runtimeError(offset, name, "variable no definida '%s'.", name)
			}
			vm.push(val)

		case OpSetGlobal:
			name := chunk.Constants[vm.readByte()].(string)
			if _, ok := vm.globals[name]; !ok {
				return nil, vm.runtimeError(offset, name, "variable no definida '%s'.", name)
			}
			vm.globals[name] = vm.peek(0)

		case OpGetLocal:
			slot := int(vm.readByte())
			vm.push(vm.stack[slot])

		case OpSetLocal:
			slot := int(vm.readByte())
			vm.stack[slot] = vm.peek(0)

		case OpNot:
			vm.push(!isTruthy(vm.pop()))

		case OpEqual:
			right, left := vm.pop(), vm.pop()
			vm.push(isEqual(left, right))

		case OpGreater:
			left, right, err := vm.popNumbers(offset, ">")
			if err != nil {
				return nil, err
			}
			vm.push(left > right)

		case OpGreaterEqual:
			left, right, err := vm.popNumbers(offset, ">=")
			if err != nil {
				return nil, err
			}
			vm.push(left >= right)

		case OpLess:
			left, right, err := vm.popNumbers(offset, "<")
			if err != nil {
				return nil, err
			}
			vm.push(left < right)

		case OpLessEqual:
			left, right, err := vm.popNumbers(offset, "<=")
			if err != nil {
				return nil, err
			}
			vm.push(left <= right)

		case OpJump:
			offsetJump := int(vm.readShort())
			vm.ip += offsetJump

		case OpJumpIfFalse:
			offsetJump := int(vm.readShort())
			if !isTruthy(vm.peek(0)) {
				vm.ip += offsetJump
			}

		case OpJumpIfTrue:
			offsetJump := int(vm.readShort())
			if isTruthy(vm.peek(0)) {
				vm.ip += offsetJump
			}

		case OpLoop:
			offsetJump := int(vm.readShort())
			vm.ip -= offsetJump

		case OpReturn:
			if len(vm.stack) == 0 {
				return nil, nil
			}
			return vm.pop(), nil

		default:
			return nil, fmt.Errorf("la VM encontró un opcode desconocido (%d) en el offset %d", byte(op), offset)
		}
	}

	return nil, fmt.Errorf("el chunk terminó sin un %s", OpReturn)
}

func (vm *VM) readByte() byte {
	b := vm.chunk.Code[vm.ip]
	vm.ip++
	return b
}

func (vm *VM) readShort() uint16 {
	high := uint16(vm.chunk.Code[vm.ip])
	low := uint16(vm.chunk.Code[vm.ip+1])
	vm.ip += 2
	return (high << 8) | low
}

func isTruthy(val Value) bool {
	if val == nil {
		return false
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return true
}

func isEqual(a, b Value) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a == b
}

func (vm *VM) push(value Value) {
	vm.stack = append(vm.stack, value)
}

func (vm *VM) pop() Value {
	top := len(vm.stack) - 1
	value := vm.stack[top]
	vm.stack = vm.stack[:top]
	return value
}

// peek devuelve el valor que está a `distance` posiciones por debajo del tope
// de la pila (peek(0) es el tope) sin desapilarlo.
func (vm *VM) peek(distance int) Value {
	return vm.stack[len(vm.stack)-1-distance]
}

// popNumbers desapila los dos operandos de una operación binaria (primero el
// derecho, que es el que quedó arriba) y verifica que ambos sean números.
// Regla de oro de Lox: los tipos se chequean recién acá, con los dos operandos
// ya evaluados.
func (vm *VM) popNumbers(offset int, operator string) (float64, float64, error) {
	right, left := vm.pop(), vm.pop()
	l, lOk := left.(float64)
	r, rOk := right.(float64)
	if !lOk || !rOk {
		return 0, 0, vm.runtimeError(offset, operator, "los operandos de '%s' deben ser números, se obtuvo: %v y %v", operator, formatValue(left), formatValue(right))
	}
	return l, r, nil
}

func (vm *VM) runtimeError(offset int, operator, format string, args ...any) error {
	return &RuntimeError{
		Line:     vm.chunk.Lines[offset],
		Operator: operator,
		Message:  fmt.Sprintf(format, args...),
	}
}
