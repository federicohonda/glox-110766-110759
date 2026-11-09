package bytecode_test

import (
	"strings"
	"testing"
)

func TestVMHerencia(t *testing.T) {
	tests := []struct {
		name, source, expected string
	}{
		{
			name:     "hereda un método",
			source:   "class A { hola() { return \"hola\"; } }\nclass B < A {}\nprint B().hola();",
			expected: "hola\n",
		},
		{
			name:     "override",
			source:   "class A { m() { return \"A\"; } }\nclass B < A { m() { return \"B\"; } }\nprint A().m();\nprint B().m();",
			expected: "A\nB\n",
		},
		{
			name:     "super llama al método pisado",
			source:   "class A { m() { return \"A\"; } }\nclass B < A { m() { return \"B+\" + super.m(); } }\nprint B().m();",
			expected: "B+A\n",
		},
		{
			// El super de B tiene que ir a A aunque this sea una C: se resuelve
			// por la clase donde está escrito el método, no por la de la instancia.
			name: "super en tres niveles",
			source: "class A { m() { return \"A\"; } }\n" +
				"class B < A { m() { return \"B\" + super.m(); } }\n" +
				"class C < B { m() { return \"C\" + super.m(); } }\n" +
				"print C().m();",
			expected: "CBA\n",
		},
		{
			name:     "método heredado de dos niveles arriba",
			source:   "class A { m() { return 1; } }\nclass B < A {}\nclass C < B {}\nprint C().m();",
			expected: "1\n",
		},
		{
			name:     "init heredado",
			source:   "class A { init(x) { this.x = x; } }\nclass B < A {}\nprint B(7).x;",
			expected: "7\n",
		},
		{
			name:     "super.init",
			source:   "class A { init(x) { this.x = x; } }\nclass B < A { init(x, y) { super.init(x); this.y = y; } }\nvar b = B(1, 2);\nprint b.x + b.y;",
			expected: "3\n",
		},
		{
			name:     "super sin llamar devuelve un método ligado a this",
			source:   "class A { m() { return this.n; } }\nclass B < A { f() { return super.m; } }\nvar b = B();\nb.n = 5;\nvar g = b.f();\nprint g();",
			expected: "5\n",
		},
		{
			name:     "super dentro de una función anidada en el método",
			source:   "class A { m() { return \"A\"; } }\nclass B < A { m() { fun g() { return super.m(); } return g(); } }\nprint B().m();",
			expected: "A\n",
		},
		{
			name:     "this en el método de la superclase es la instancia de la subclase",
			source:   "class A { nombre() { return this.quien(); } }\nclass B < A { quien() { return \"B\"; } }\nprint B().nombre();",
			expected: "B\n",
		},
		{
			name:     "la superclase no ve los métodos de la subclase",
			source:   "class A {}\nclass B < A { m() {} }\nprint A().m;",
			expected: "",
		},
		{
			name:     "subclase declarada en un bloque",
			source:   "{\n  class A { m() { return 1; } }\n  class B < A { m() { return super.m() + 1; } }\n  print B().m();\n}",
			expected: "2\n",
		},
		{
			name:     "superclase local capturada desde una función",
			source:   "fun crear() {\n  class A { m() { return \"A\"; } }\n  class B < A { m() { return super.m() + \"B\"; } }\n  return B;\n}\nprint crear()().m();",
			expected: "AB\n",
		},
		{
			name:     "las instancias de la subclase siguen teniendo su propio estado",
			source:   "class A { init() { this.n = 0; } sumar() { this.n = this.n + 1; return this.n; } }\nclass B < A {}\nvar x = B();\nvar y = B();\nx.sumar();\nx.sumar();\nprint x.sumar();\nprint y.sumar();",
			expected: "3\n1\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expected == "" {
				if _, err := runProgram(t, tt.source); err == nil || !strings.Contains(err.Error(), "propiedad no definida 'm'") {
					t.Fatalf("error = %v, se esperaba propiedad no definida 'm'", err)
				}
				return
			}
			if got := mustRunProgram(t, tt.source); got != tt.expected {
				t.Errorf("salida = %q, esperado %q", got, tt.expected)
			}
		})
	}
}

// El listado de una subclase con super es el que quedó diseñado en el plan
// antes de escribir el código: la superclase queda como local `super` en una
// celda, y el método la captura.
func TestDisassembleHerencia(t *testing.T) {
	chunk := mustCompileProgram(t, "class A { m() { return 1; } }\nclass B < A {\n  m() { return super.m(); }\n}")
	dis := chunk.Disassemble("programa")
	for _, want := range []string{
		"0015    | OP_GET_GLOBAL       0 'A'\n0017    | OP_GET_GLOBAL       3 'B'\n0019    | OP_INHERIT\n0020    | OP_BOX              0\n",
		"OP_CLOSURE          4 '<fn m>'\n     |                     local 0\n",
		"0030    2 OP_POP\n0031    | OP_POP\n",
		"== <fn m> ==\n0000    3 OP_GET_LOCAL        0\n0002    | OP_GET_CAPTURED     0\n0004    | OP_SUPER_INVOKE  (0 args)    0 'm'\n",
	} {
		if !strings.Contains(dis, want) {
			t.Errorf("el listado no contiene\n%s\nlistado:\n%s", want, dis)
		}
	}
}

func TestVMErroresHerencia(t *testing.T) {
	tests := []struct {
		name, source, expected string
	}{
		{
			name:     "la superclase no es una clase",
			source:   "var A = 3;\nclass B < A {}",
			expected: "[línea 2] Error en tiempo de ejecución en '<': la superclase debe ser una clase, se obtuvo: 3",
		},
		{
			name:     "super.m que no existe",
			source:   "class A {}\nclass B < A { m() { return super.m(); } }\nB().m();",
			expected: "[línea 2] Error en tiempo de ejecución en 'm': propiedad no definida 'm'.",
		},
		{
			name:     "super.m sin llamar que no existe",
			source:   "class A {}\nclass B < A { m() { return super.m; } }\nB().m();",
			expected: "[línea 2] Error en tiempo de ejecución en 'm': propiedad no definida 'm'.",
		},
		{
			name:     "aridad del init heredado",
			source:   "class A { init(x) {} }\nclass B < A {}\nB();",
			expected: "[línea 3] Error en tiempo de ejecución en ')': se esperaban 1 argumentos pero se recibieron 0.",
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
