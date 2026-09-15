package interpreter

import (
	"fmt"

	"github.com/federicohonda/glox-110766-110759/internal/ast"
)

// Callable es cualquier valor de Lox que se puede invocar con `valor(args)`.
type Callable interface {
	Call(interp *Interpreter, args []Value) (Value, error)
	Arity() int
}

// Function es una función declarada por el usuario con `fun`. Closure es el
// entorno vigente en el momento de DECLARARLA (no el de invocarla): eso es lo
// que permite que la función siga viendo las variables de ese scope aunque
// ya se haya salido de él.
type Function struct {
	Declaration *ast.FunDecl
	Closure     *Environment
}

func (f *Function) Arity() int {
	return len(f.Declaration.Params)
}

// Call ejecuta el cuerpo en un entorno nuevo por invocación, cuyo padre es la
// closure. Que sea nuevo en cada llamada es lo que hace que la recursión y
// las llamadas repetidas no compartan parámetros ni variables locales.
func (f *Function) Call(interp *Interpreter, args []Value) (Value, error) {
	env := NewEnclosingEnvironment(f.Closure)
	for i, param := range f.Declaration.Params {
		env.Define(param.Lexeme, args[i])
	}

	if err := interp.executeBlock(f.Declaration.Body, env); err != nil {
		return nil, err
	}
	return nil, nil
}

func (f *Function) String() string {
	return fmt.Sprintf("<fn %s>", f.Declaration.Name.Lexeme)
}
