package bytecode

import (
	"fmt"
	"math"
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
	ip    int
	stack []Value
}

// NewVM crea una VM con la pila vacía.
func NewVM() *VM {
	return &VM{stack: make([]Value, 0, stackInitialCapacity)}
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

		case OpReturn:
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

func (vm *VM) push(value Value) {
	vm.stack = append(vm.stack, value)
}

func (vm *VM) pop() Value {
	top := len(vm.stack) - 1
	value := vm.stack[top]
	vm.stack = vm.stack[:top]
	return value
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
