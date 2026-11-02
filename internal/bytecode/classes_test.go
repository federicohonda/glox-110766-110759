package bytecode_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/federicohonda/glox-110766-110759/internal/bytecode"
)

func TestVMClasesBasicas(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{
			name:     "imprimir clase",
			source:   "class DevonshireCream {}\nprint DevonshireCream;",
			expected: "DevonshireCream\n",
		},
		{
			name:     "imprimir instancia",
			source:   "class Bagel {}\nvar b = Bagel();\nprint b;",
			expected: "Bagel instance\n",
		},
		{
			name:     "asignar y leer propiedades dinámicas",
			source:   "class Box {}\nvar b = Box();\nb.item = \"gloves\";\nprint b.item;\nb.item = \"hat\";\nprint b.item;",
			expected: "gloves\nhat\n",
		},
		{
			name:     "múltiples propiedades e instancias independientes",
			source:   "class Pair {}\nvar p1 = Pair();\nvar p2 = Pair();\np1.a = 1;\np2.a = 2;\nprint p1.a;\nprint p2.a;",
			expected: "1\n2\n",
		},
		{
			name:     "método simple con OpInvoke",
			source:   "class Bacon {\n  eat() {\n    print \"Crunch crunch!\";\n  }\n}\nBacon().eat();",
			expected: "Crunch crunch!\n",
		},
		{
			name:     "método que usa this",
			source:   "class Cake {\n  taste() {\n    print \"Delicious \" + this.flavor + \" cake!\";\n  }\n}\nvar c = Cake();\nc.flavor = \"chocolate\";\nc.taste();",
			expected: "Delicious chocolate cake!\n",
		},
		{
			name:     "método llamando a otro método en this",
			source:   "class Greeter {\n  name() {\n    return \"Lox\";\n  }\n  greet() {\n    print \"Hello \" + this.name() + \"!\";\n  }\n}\nGreeter().greet();",
			expected: "Hello Lox!\n",
		},
		{
			name:     "método ligado almacenado en variable",
			source:   "class Person {\n  say() {\n    print this.name;\n  }\n}\nvar p = Person();\np.name = \"Alice\";\nvar fn = p.say;\nfn();",
			expected: "Alice\n",
		},
		{
			name:     "campo sombrea a método",
			source:   "class Foo {\n  bar() {\n    return \"metodo\";\n  }\n}\nvar f = Foo();\nf.bar = \"campo\";\nprint f.bar;",
			expected: "campo\n",
		},
		{
			name:     "campo con función invocable como método",
			source:   "class Box {}\nfun saludo(x) {\n  print \"hola \" + x;\n}\nvar b = Box();\nb.run = saludo;\nb.run(\"mundo\");",
			expected: "hola mundo\n",
		},
		{
			name:     "inicializador con parámetros",
			source:   "class Point {\n  init(x, y) {\n    this.x = x;\n    this.y = y;\n  }\n}\nvar p = Point(10, 20);\nprint p.x + p.y;",
			expected: "30\n",
		},
		{
			name:     "inicializador con return vacío temprano devuelve this",
			source:   "class Check {\n  init(val) {\n    this.val = val;\n    if (val < 0) return;\n    this.positive = true;\n  }\n}\nvar c = Check(-5);\nprint c.val;",
			expected: "-5\n",
		},
		{
			name:     "closure dentro de método captura this",
			source:   "class Multiplier {\n  init(base) {\n    this.base = base;\n  }\n  getMultiplier() {\n    fun mult(x) {\n      return this.base * x;\n    }\n    return mult;\n  }\n}\nvar m = Multiplier(3).getMultiplier();\nprint m(7);",
			expected: "21\n",
		},
		{
			name:     "clase declarada en bloque local",
			source:   "{\n  class Local {\n    msg() { return \"local\"; }\n  }\n  print Local().msg();\n}",
			expected: "local\n",
		},
		{
			name:     "método recursivo que referencia el nombre de su clase",
			source:   "class Node {\n  init(val) {\n    this.val = val;\n  }\n  clone() {\n    return Node(this.val);\n  }\n}\nvar n = Node(42).clone();\nprint n.val;",
			expected: "42\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mustRunProgram(t, tt.source)
			if got != tt.expected {
				t.Errorf("salida = %q, esperado %q", got, tt.expected)
			}
		})
	}
}

func TestVMDisassembleClases(t *testing.T) {
	source := `class Math {
  square(x) {
    return x * x;
  }
}
var m = Math();
m.square(4);
`
	chunk := mustCompileProgram(t, source)
	dis := chunk.Disassemble("programa")
	for _, expectedOp := range []string{"OP_CLASS", "OP_METHOD", "OP_INVOKE"} {
		if !strings.Contains(dis, expectedOp) {
			t.Errorf("el desmontaje debía contener %q pero no lo tiene:\n%s", expectedOp, dis)
		}
	}
}

func TestVMErroresRuntimeClases(t *testing.T) {
	tests := []struct {
		name, source, expected string
	}{
		{
			name:     "acceder propiedad en número",
			source:   "var n = 123;\nprint n.foo;",
			expected: "[línea 2] Error en tiempo de ejecución en '.': solo las instancias tienen propiedades.",
		},
		{
			name:     "asignar propiedad en número",
			source:   "var n = 123;\nn.foo = 1;",
			expected: "[línea 2] Error en tiempo de ejecución en '.': solo las instancias tienen propiedades.",
		},
		{
			name:     "propiedad no definida en instancia",
			source:   "class Empty {}\nvar e = Empty();\nprint e.inexistente;",
			expected: "[línea 3] Error en tiempo de ejecución en 'inexistente': propiedad no definida 'inexistente'.",
		},
		{
			name:     "invocar método en número",
			source:   "var n = 123;\nn.foo();",
			expected: "[línea 2] Error en tiempo de ejecución en '.': solo las instancias tienen métodos.",
		},
		{
			name:     "invocar método no definido en instancia",
			source:   "class Empty {}\nEmpty().inexistente();",
			expected: "[línea 2] Error en tiempo de ejecución en 'inexistente': propiedad no definida 'inexistente'.",
		},
		{
			name:     "argumentos en clase sin init",
			source:   "class Foo {}\nFoo(1);",
			expected: "[línea 2] Error en tiempo de ejecución en ')': se esperaban 0 argumentos pero se recibieron 1.",
		},
		{
			name:     "aridad incorrecta en init",
			source:   "class Point {\n  init(x, y) {}\n}\nPoint(1);",
			expected: "[línea 4] Error en tiempo de ejecución en ')': se esperaban 2 argumentos pero se recibieron 1.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runProgram(t, tt.source)
			var runtimeErr *bytecode.RuntimeError
			if !errors.As(err, &runtimeErr) {
				t.Fatalf("error = %v, se esperaba un RuntimeError", err)
			}
			if err.Error() != tt.expected {
				t.Errorf("mensaje = %q\nesperado  %q", err.Error(), tt.expected)
			}
		})
	}
}
