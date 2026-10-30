package bytecode

import "fmt"

// Function es una función ya compilada: su propio chunk más lo que la VM
// necesita saber para invocarla. Es un valor de tiempo de compilación (vive
// en la tabla de constantes del chunk que la declara); en runtime lo que se
// pasa de un lado a otro es la Closure que la envuelve.
type Function struct {
	Name  string
	Arity int
	Chunk *Chunk
	// CapturedCount es cuántas variables captura: el largo de Closure.Captured
	// y la cantidad de pares de operandos que lleva su OpClosure.
	CapturedCount int
}

func (f *Function) String() string {
	return fmt.Sprintf("<fn %s>", f.Name)
}

// Closure es el valor de runtime de una función: el código compilado (uno
// solo, compartido por todas las closures de esa función) más las variables
// que capturó esta closure en particular.
type Closure struct {
	Function *Function
	Captured []*Cell
}

func (c *Closure) String() string {
	return c.Function.String()
}

// Cell es una variable local capturada por alguna función anidada. Vive en el
// heap desde que se declara: el slot de la pila guarda la celda, no el valor,
// así la variable sobrevive a su frame sin tener que mudarla al retornar.
type Cell struct {
	Value Value
}
