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

// Class es el valor de runtime de una clase: su nombre y su tabla de métodos.
type Class struct {
	Name    string
	Methods map[string]*Closure
}

func (c *Class) String() string {
	return c.Name
}

// Instance es una instancia concreta de una Class: referencia a su clase
// y una tabla de campos (atributos) dinámicos.
type Instance struct {
	Class  *Class
	Fields map[string]Value
}

func (i *Instance) String() string {
	return fmt.Sprintf("%s instance", i.Class.Name)
}

// BoundMethod es un método ligado a su instancia receptora: cuando se invoca,
// el receiver se coloca en el slot 0 ('this') del frame de la llamada.
type BoundMethod struct {
	Receiver *Instance
	Method   *Closure
}

func (b *BoundMethod) String() string {
	return b.Method.String()
}
