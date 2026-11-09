package bytecode

import (
	"fmt"
	"unicode/utf8"
)

// natives son las funciones que la VM define como globales al arrancar. Son
// las mínimas para poder usar las listas: sin len no hay forma de recorrerlas,
// y sin push no hay forma de hacerlas crecer.
var natives = []*Native{
	{Name: "len", Arity: 1, Fn: nativeLen},
	{Name: "push", Arity: 2, Fn: nativePush},
}

// nativeLen devuelve la cantidad de elementos de una lista o de caracteres de
// un string (caracteres, no bytes: len("año") es 3).
func nativeLen(args []Value) (Value, error) {
	switch v := args[0].(type) {
	case *List:
		return float64(len(v.Items)), nil
	case string:
		return float64(utf8.RuneCountInString(v)), nil
	}
	return nil, fmt.Errorf("len espera una lista o un string, se obtuvo: %v", formatValue(args[0]))
}

// nativePush agrega un elemento al final de la lista y devuelve nil.
func nativePush(args []Value) (Value, error) {
	list, ok := args[0].(*List)
	if !ok {
		return nil, fmt.Errorf("push espera una lista como primer argumento, se obtuvo: %v", formatValue(args[0]))
	}
	list.Items = append(list.Items, args[1])
	return nil, nil
}
