package bytecode_test

import (
	"strings"
	"testing"
)

func TestVMListas(t *testing.T) {
	tests := []struct {
		name, source, expected string
	}{
		{"lista vacía", "print [];", "[]\n"},
		{"se imprime como print", "print [1, \"dos\", nil, true, [3]];", "[1, dos, nil, true, [3]]\n"},
		{"leer por índice", "var a = [10, 20, 30];\nprint a[0] + a[2];", "40\n"},
		{"asignar por índice", "var a = [1, 2];\na[1] = \"x\";\nprint a;", "[1, x]\n"},
		{"la asignación es una expresión", "var a = [0, 0];\nprint a[0] = a[1] = 5;\nprint a;", "5\n[5, 5]\n"},
		{"listas anidadas", "var m = [[1, 2], [3, 4]];\nm[1][0] = 9;\nprint m[1][0] + m[0][1];", "11\n"},
		{"len de una lista", "print len([]);\nprint len([1, 2, 3]);", "0\n3\n"},
		{"len de un string cuenta caracteres", "print len(\"año\");", "3\n"},
		{"push agrega al final", "var a = [];\npush(a, 1);\npush(a, 2);\nprint a;\nprint len(a);", "[1, 2]\n2\n"},
		{"push devuelve nil", "print push([], 1);", "nil\n"},
		{"las listas son referencias", "var a = [1];\nvar b = a;\npush(b, 2);\nprint a;", "[1, 2]\n"},
		{"== compara identidad", "var a = [1];\nprint a == a;\nprint [1] == [1];", "true\nfalse\n"},
		{"una lista que se contiene a sí misma", "var a = [1];\npush(a, a);\nprint a;", "[1, [...]]\n"},
		{"recorrer con for", "var a = [1, 2, 3];\nvar s = 0;\nfor (var i = 0; i < len(a); i = i + 1) s = s + a[i];\nprint s;", "6\n"},
		{"lista local capturada por una closure", "fun pila() {\n  var items = [];\n  fun apilar(x) { push(items, x); return items; }\n  return apilar;\n}\nvar p = pila();\np(1);\nprint p(2);", "[1, 2]\n"},
		{"lista como campo de una instancia", "class Bolsa { init() { this.items = []; } agregar(x) { push(this.items, x); } }\nvar b = Bolsa();\nb.agregar(\"a\");\nb.agregar(\"b\");\nprint b.items[1];", "b\n"},
		{"índice calculado", "var a = [\"a\", \"b\", \"c\"];\nprint a[len(a) - 1];", "c\n"},
		{"las nativas son valores", "var f = len;\nprint f;\nprint f([1]);", "<fn nativa len>\n1\n"},
		{"una nativa se puede tapar", "fun len(x) { return 42; }\nprint len([]);", "42\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mustRunProgram(t, tt.source); got != tt.expected {
				t.Errorf("salida = %q, esperado %q", got, tt.expected)
			}
		})
	}
}

func TestDisassembleListas(t *testing.T) {
	dis := mustCompileProgram(t, "var a = [1, 2];\na[0] = a[1];").Disassemble("programa")
	for _, want := range []string{"OP_LIST             2", "OP_GET_INDEX", "OP_SET_INDEX"} {
		if !strings.Contains(dis, want) {
			t.Errorf("el listado no contiene %q:\n%s", want, dis)
		}
	}
}

func TestVMErroresListas(t *testing.T) {
	tests := []struct {
		name, source, expected string
	}{
		{"índice fuera de rango", "var a = [1, 2];\nprint a[2];", "[línea 2] Error en tiempo de ejecución en '[': índice fuera de rango: 2 (la lista tiene 2 elementos)."},
		{"índice negativo", "var a = [1];\nprint a[-1];", "[línea 2] Error en tiempo de ejecución en '[': índice fuera de rango: -1 (la lista tiene 1 elementos)."},
		{"índice con decimales", "print [1][0.5];", "[línea 1] Error en tiempo de ejecución en '[': el índice debe ser un número entero, se obtuvo: 0.5"},
		{"índice que no es número", "print [1][\"0\"];", "[línea 1] Error en tiempo de ejecución en '[': el índice debe ser un número entero, se obtuvo: 0"},
		{"indexar algo que no es lista", "var s = \"hola\";\nprint s[0];", "[línea 2] Error en tiempo de ejecución en '[': solo se pueden indexar listas, se obtuvo: hola"},
		{"asignar fuera de rango", "var a = [];\na[0] = 1;", "[línea 2] Error en tiempo de ejecución en '[': índice fuera de rango: 0 (la lista tiene 0 elementos)."},
		{"len de un número", "print len(3);", "[línea 1] Error en tiempo de ejecución en 'len': len espera una lista o un string, se obtuvo: 3"},
		{"push sobre algo que no es lista", "push(nil, 1);", "[línea 1] Error en tiempo de ejecución en 'push': push espera una lista como primer argumento, se obtuvo: nil"},
		{"aridad de una nativa", "push([]);", "[línea 1] Error en tiempo de ejecución en ')': se esperaban 2 argumentos pero se recibieron 1."},
		{
			// Regla de oro: se evalúan lista, índice y valor antes de chequear nada.
			"el valor se evalúa antes de chequear el índice",
			"var a = [];\na[5] = 1 / 0;",
			"[línea 2] Error en tiempo de ejecución en '/': división por cero.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runProgram(t, tt.source)
			if err == nil || err.Error() != tt.expected {
				t.Errorf("error = %v\nesperado  %s", err, tt.expected)
			}
		})
	}
}
