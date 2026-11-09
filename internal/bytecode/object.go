package bytecode

import (
	"fmt"
	"strings"
)

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

// List es una lista de Lox: un slice de Go que crece con push. Va siempre
// por puntero, así dos variables que apuntan a la misma lista ven los mismos
// cambios y `==` compara identidad, igual que entre instancias.
type List struct {
	Items []Value
}

func (l *List) String() string {
	return l.format(map[*List]bool{})
}

// format imprime los elementos como los imprimiría print. seen corta el caso
// de una lista que se contiene a sí misma, que si no recursaría para siempre.
func (l *List) format(seen map[*List]bool) string {
	if seen[l] {
		return "[...]"
	}
	seen[l] = true
	defer delete(seen, l)
	parts := make([]string, len(l.Items))
	for i, item := range l.Items {
		if inner, ok := item.(*List); ok {
			parts[i] = inner.format(seen)
		} else {
			parts[i] = formatValue(item)
		}
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// Native es una función de la biblioteca estándar, escrita en Go. La VM la
// llama directo, sin abrir un frame: recibe los argumentos que están en la
// pila y devuelve el resultado o un error (que la VM convierte en un error de
// runtime con la línea de la llamada).
type Native struct {
	Name  string
	Arity int
	Fn    func(args []Value) (Value, error)
}

func (n *Native) String() string {
	return fmt.Sprintf("<fn nativa %s>", n.Name)
}
