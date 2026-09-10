package interpreter

import (
	"fmt"

	"github.com/federicohonda/glox-110766-110759/internal/token"
)

// Environment almacena las vinculaciones (bindings) de variables a valores
// y mantiene una referencia al entorno que lo encierra (enclosing) para
// implementar el scoping léxico anidado.
type Environment struct {
	values    map[string]Value
	enclosing *Environment
}

// NewEnvironment crea un nuevo entorno sin entorno padre (entorno raíz/global).
func NewEnvironment() *Environment {
	return &Environment{
		values: make(map[string]Value),
	}
}

// NewEnclosingEnvironment crea un nuevo entorno anidado dentro de enclosing.
func NewEnclosingEnvironment(enclosing *Environment) *Environment {
	return &Environment{
		values:    make(map[string]Value),
		enclosing: enclosing,
	}
}

// Define registra o redefine una variable en el ámbito local actual.
// En Lox, se permite la redeclaración de variables en el mismo ámbito.
func (e *Environment) Define(name string, value Value) {
	e.values[name] = value
}

// Get busca el valor asociado al identificador. Si no existe en el ámbito
// actual, busca recursivamente hacia los ámbitos padres (enclosing).
// Si no existe en ningún nivel, produce un RuntimeError.
func (e *Environment) Get(name token.Token) (Value, error) {
	if val, ok := e.values[name.Lexeme]; ok {
		return val, nil
	}

	if e.enclosing != nil {
		return e.enclosing.Get(name)
	}

	return nil, &RuntimeError{
		Token:   name,
		Message: fmt.Sprintf("variable no definida '%s'.", name.Lexeme),
	}
}

// Assign asigna un nuevo valor a una variable ya declarada.
// Primero busca en el ámbito actual; si no existe, busca en los ámbitos padres.
// Si no se encuentra en ningún nivel, produce un RuntimeError.
func (e *Environment) Assign(name token.Token, value Value) error {
	if _, ok := e.values[name.Lexeme]; ok {
		e.values[name.Lexeme] = value
		return nil
	}

	if e.enclosing != nil {
		return e.enclosing.Assign(name, value)
	}

	return &RuntimeError{
		Token:   name,
		Message: fmt.Sprintf("variable no definida '%s'.", name.Lexeme),
	}
}
