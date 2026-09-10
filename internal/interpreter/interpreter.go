package interpreter

import (
	"fmt"
	"math"

	"github.com/federicohonda/glox-110766-110759/internal/ast"
	"github.com/federicohonda/glox-110766-110759/internal/token"
)

// Value representa cualquier valor en tiempo de ejecución de Lox:
// float64, string, bool, nil (o funciones/clases en fases posteriores).
type Value = any

// RuntimeError representa un error producido durante la evaluación de una
// expresión (por ejemplo, tipos incompatibles o división por cero).
// Contiene el token donde ocurrió para reportar la línea y contexto.
type RuntimeError struct {
	Token   token.Token
	Message string
}

func (e *RuntimeError) Error() string {
	if e.Token.Type == token.EOF {
		return fmt.Sprintf("[línea %d] Error en tiempo de ejecución: %s", e.Token.Line, e.Message)
	}
	return fmt.Sprintf("[línea %d] Error en tiempo de ejecución en '%s': %s", e.Token.Line, e.Token.Lexeme, e.Message)
}

// Interpreter evalúa nodos del AST y produce valores o errores de runtime.
// El despacho por tipo de nodo se realiza mediante un type switch idiomático de Go.
type Interpreter struct{}

// New crea una nueva instancia de Interpreter.
func New() *Interpreter {
	return &Interpreter{}
}

// Evaluate despacha la evaluación de la expresión según su tipo concreto de nodo.
func (i *Interpreter) Evaluate(expr ast.Expr) (Value, error) {
	switch e := expr.(type) {
	case *ast.Literal:
		return e.Value, nil

	case *ast.Grouping:
		return i.Evaluate(e.Expression)

	case *ast.Unary:
		right, err := i.Evaluate(e.Right)
		if err != nil {
			return nil, err
		}

		switch e.Operator.Type {
		case token.MINUS:
			r, ok := right.(float64)
			if !ok {
				return nil, &RuntimeError{
					Token:   e.Operator,
					Message: fmt.Sprintf("el operando de '-' debe ser un número, se obtuvo: %v", Stringify(right)),
				}
			}
			return -r, nil

		case token.BANG:
			return !isTruthy(right), nil

		default:
			return nil, &RuntimeError{
				Token:   e.Operator,
				Message: fmt.Sprintf("operador unario desconocido: %s", e.Operator.Lexeme),
			}
		}

	case *ast.Binary:
		// Regla de oro de Lox: evaluar primero ambos operandos (izquierdo y luego derecho)
		// antes de verificar tipos o ejecutar la operación, asegurando que cualquier
		// efecto colateral ocurra en el orden esperado.
		left, err := i.Evaluate(e.Left)
		if err != nil {
			return nil, err
		}

		right, err := i.Evaluate(e.Right)
		if err != nil {
			return nil, err
		}

		switch e.Operator.Type {
		case token.PLUS:
			switch l := left.(type) {
			case float64:
				if r, ok := right.(float64); ok {
					return l + r, nil
				}
			case string:
				if r, ok := right.(string); ok {
					return l + r, nil
				}
			}
			return nil, &RuntimeError{
				Token:   e.Operator,
				Message: fmt.Sprintf("los operandos de '+' deben ser dos números o dos cadenas, se obtuvo: %v y %v", Stringify(left), Stringify(right)),
			}

		case token.MINUS:
			l, r, err := checkNumberOperands(e.Operator, left, right)
			if err != nil {
				return nil, err
			}
			return l - r, nil

		case token.STAR:
			l, r, err := checkNumberOperands(e.Operator, left, right)
			if err != nil {
				return nil, err
			}
			return l * r, nil

		case token.SLASH:
			l, r, err := checkNumberOperands(e.Operator, left, right)
			if err != nil {
				return nil, err
			}
			if r == 0 {
				return nil, &RuntimeError{
					Token:   e.Operator,
					Message: "división por cero.",
				}
			}
			return l / r, nil

		case token.PERCENT:
			l, r, err := checkNumberOperands(e.Operator, left, right)
			if err != nil {
				return nil, err
			}
			if r == 0 {
				return nil, &RuntimeError{
					Token:   e.Operator,
					Message: "módulo por cero.",
				}
			}
			return math.Mod(l, r), nil

		case token.GREATER:
			l, r, err := checkNumberOperands(e.Operator, left, right)
			if err != nil {
				return nil, err
			}
			return l > r, nil

		case token.GREATER_EQUAL:
			l, r, err := checkNumberOperands(e.Operator, left, right)
			if err != nil {
				return nil, err
			}
			return l >= r, nil

		case token.LESS:
			l, r, err := checkNumberOperands(e.Operator, left, right)
			if err != nil {
				return nil, err
			}
			return l < r, nil

		case token.LESS_EQUAL:
			l, r, err := checkNumberOperands(e.Operator, left, right)
			if err != nil {
				return nil, err
			}
			return l <= r, nil

		case token.EQUAL_EQUAL:
			return isEqual(left, right), nil

		case token.BANG_EQUAL:
			return !isEqual(left, right), nil

		default:
			return nil, &RuntimeError{
				Token:   e.Operator,
				Message: fmt.Sprintf("operador binario desconocido: %s", e.Operator.Lexeme),
			}
		}

	default:
		return nil, fmt.Errorf("tipo de expresión no soportado: %T", expr)
	}
}

// checkNumberOperands valida que ambos operandos sean float64.
func checkNumberOperands(operator token.Token, left, right Value) (float64, float64, error) {
	l, lOk := left.(float64)
	r, rOk := right.(float64)
	if !lOk || !rOk {
		return 0, 0, &RuntimeError{
			Token:   operator,
			Message: fmt.Sprintf("los operandos de '%s' deben ser números, se obtuvo: %v y %v", operator.Lexeme, Stringify(left), Stringify(right)),
		}
	}
	return l, r, nil
}

// isTruthy implementa la semántica de veracidad de Lox (estilo Ruby):
// nil y false son falsos; cualquier otro valor es verdadero (incluso 0 y "").
func isTruthy(val Value) bool {
	if val == nil {
		return false
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return true
}

// isEqual compara dos valores de Lox sin coerción de tipos.
// nil solo es igual a nil.
func isEqual(a, b Value) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a == b
}

// Stringify formatea un Value como string para mostrar al usuario.
func Stringify(val Value) string {
	if val == nil {
		return "nil"
	}
	return fmt.Sprintf("%v", val)
}
