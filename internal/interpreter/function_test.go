package interpreter_test

import (
	"strings"
	"testing"

	"github.com/federicohonda/glox-110766-110759/internal/ast"
	"github.com/federicohonda/glox-110766-110759/internal/interpreter"
	"github.com/federicohonda/glox-110766-110759/internal/token"
)

// Estos casos sacan los resultados asignando a variables de afuera, para
// probar las llamadas y las closures sin depender de `return`. Igual que en el
// resto de los tests, si algo no da lo esperado se fuerza un error con
// `print 1 / 0`.
func TestInterpretFunctionCalls(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{
			name: "llamada con parámetros",
			source: `
				var resultado;
				fun suma(a, b) { resultado = a + b; }
				suma(1, 2);
				if (resultado != 3) print 1 / 0;
			`,
		},
		{
			name: "función sin parámetros",
			source: `
				var llamada = false;
				fun marcar() { llamada = true; }
				marcar();
				if (!llamada) print 1 / 0;
			`,
		},
		{
			name: "una llamada sin return evalúa a nil",
			source: `
				fun nada() {}
				if (nada() != nil) print 1 / 0;
			`,
		},
		{
			name: "los parámetros no se ven fuera de la función",
			source: `
				var x = "global";
				fun f(x) { x = "parámetro"; }
				f("argumento");
				if (x != "global") print 1 / 0;
			`,
		},
		{
			name: "recursión: cada llamada tiene su propio entorno",
			source: `
				var suma = 0;
				fun acumular(n) {
					if (n > 0) {
						acumular(n - 1);
						suma = suma + n;
					}
				}
				acumular(10);
				if (suma != 55) print 1 / 0;
			`,
		},
		{
			name: "función declarada dentro de un bloque",
			source: `
				var visto;
				{
					fun f() { visto = "bloque"; }
					f();
				}
				if (visto != "bloque") print 1 / 0;
			`,
		},
		{
			name: "una función es un valor que se puede asignar",
			source: `
				var resultado;
				fun f(x) { resultado = x; }
				var alias = f;
				alias(7);
				if (resultado != 7) print 1 / 0;
				if (alias != f) print 1 / 0;
			`,
		},
		{
			name: "los argumentos se evalúan en orden, de izquierda a derecha",
			source: `
				var orden = "";
				fun marca(s) { orden = orden + s; }
				fun tres(a, b, c) {}
				tres(marca("a"), marca("b"), marca("c"));
				if (orden != "abc") print 1 / 0;
			`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := interpretSource(t, c.source); err != nil {
				t.Fatalf("interpretSource falló: %v", err)
			}
		})
	}
}

// Checkpoint del plan: una función sigue viendo las variables del entorno
// donde fue declarada, aunque ese entorno ya haya "salido de scope".
func TestInterpretClosures(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{
			name: "lee una variable de un bloque que ya terminó",
			source: `
				var resultado;
				var g;
				{
					var capturada = "sigo viva";
					fun f() { resultado = capturada; }
					g = f;
				}
				g();
				if (resultado != "sigo viva") print 1 / 0;
			`,
		},
		{
			name: "contador: el estado capturado persiste entre llamadas",
			source: `
				var ultimo;
				var contar;
				{
					var i = 0;
					fun incrementar() { i = i + 1; ultimo = i; }
					contar = incrementar;
				}
				contar();
				contar();
				contar();
				if (ultimo != 3) print 1 / 0;
			`,
		},
		{
			name: "dos closures de la misma declaración no comparten estado",
			source: `
				var ultimo;
				var a;
				var b;
				fun crear(destino) {
					var i = 0;
					fun incrementar() { i = i + 1; ultimo = i; }
					if (destino == "a") a = incrementar;
					else b = incrementar;
				}
				crear("a");
				crear("b");
				a(); a(); a();
				b();
				if (ultimo != 1) print 1 / 0;
				a();
				if (ultimo != 4) print 1 / 0;
			`,
		},
		{
			name: "closures anidadas en varios niveles",
			source: `
				var resultado;
				var g;
				fun externa(x) {
					fun media(y) {
						fun interna() { resultado = x + y; }
						g = interna;
					}
					media(2);
				}
				externa(1);
				g();
				if (resultado != 3) print 1 / 0;
			`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := interpretSource(t, c.source); err != nil {
				t.Fatalf("interpretSource falló: %v", err)
			}
		})
	}
}

// Checkpoint de la Parte 4: con el resolver, una función ve siempre la
// variable que existía cuando se declaró, aunque después se redeclare otra
// con el mismo nombre en el mismo bloque.
func TestInterpretClosureBugFixed(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{
			name: "ejemplo del plan: global/global",
			source: `
				var visto = "";
				var a = "global";
				{
					fun showA() { visto = visto + a + ";"; }
					showA();
					var a = "block";
					showA();
				}
				if (visto != "global;global;") print 1 / 0;
			`,
		},
		{
			name: "CLOSURE BUG de las pruebas de la cátedra",
			source: `
				var a = "global";
				{
					fun ret_a() { return a; }
					if (ret_a() != "global") print 1 / 0;
					var a = "block";
					if (ret_a() != "global") print 1 / 0;
				}
			`,
		},
		{
			name: "la asignación también respeta la variable capturada",
			source: `
				var a = "global";
				{
					fun setA() { a = "cambiada"; }
					var a = "block";
					setA();
					if (a != "block") print 1 / 0;
				}
				if (a != "cambiada") print 1 / 0;
			`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := interpretSource(t, c.source); err != nil {
				t.Fatalf("interpretSource falló: %v", err)
			}
		})
	}
}

func TestInterpretCallErrors(t *testing.T) {
	cases := []struct {
		name    string
		source  string
		wantMsg string
	}{
		{"llamar a un número", "123();", "solo se pueden llamar funciones"},
		{"llamar a un string", `"hola"();`, "solo se pueden llamar funciones"},
		{"llamar a nil", "var f; f();", "solo se pueden llamar funciones"},
		{"menos argumentos que parámetros", "fun f(a, b) {} f(1);", "se esperaban 2 argumentos pero se recibieron 1"},
		{"más argumentos que parámetros", "fun f() {} f(1, 2);", "se esperaban 0 argumentos pero se recibieron 2"},
		{"función no definida", "noExiste();", "variable no definida"},
		{"el error dentro del cuerpo se propaga", "fun f() { print 1 / 0; } f();", "división por cero"},
		{
			// Regla de oro: el argumento se evalúa (y falla) antes de detectar
			// que el callee no es invocable.
			"los argumentos se evalúan antes de chequear el callee",
			`123(1 / 0);`,
			"división por cero",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := interpretSource(t, c.source)
			if err == nil {
				t.Fatalf("esperaba un error de runtime en %q", c.source)
			}
			if !strings.Contains(err.Error(), c.wantMsg) {
				t.Fatalf("error = %q, esperaba que contenga %q", err, c.wantMsg)
			}
		})
	}
}

func TestInterpretReturn(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{
			name: "devuelve un valor",
			source: `
				fun suma(a, b) { return a + b; }
				if (suma(1, 2) != 3) print 1 / 0;
			`,
		},
		{
			name: "checkpoint: factorial recursivo corta en el caso base",
			source: `
				fun factorial(n) {
					if (n <= 1) return 1;
					return n * factorial(n - 1);
				}
				if (factorial(1) != 1) print 1 / 0;
				if (factorial(5) != 120) print 1 / 0;
				if (factorial(10) != 3628800) print 1 / 0;
			`,
		},
		{
			name: "fibonacci recursivo",
			source: `
				fun fib(n) {
					if (n <= 1) return n;
					return fib(n - 2) + fib(n - 1);
				}
				if (fib(20) != 6765) print 1 / 0;
			`,
		},
		{
			name: "return sin valor devuelve nil",
			source: `
				fun vacio() { return; }
				if (vacio() != nil) print 1 / 0;
			`,
		},
		{
			name: "el código después del return no se ejecuta",
			source: `
				fun f() {
					return "antes";
					print 1 / 0;
				}
				if (f() != "antes") print 1 / 0;
			`,
		},
		{
			name: "atraviesa bloques anidados",
			source: `
				fun f() {
					{
						{
							{ return "profundo"; }
						}
					}
					return "no debería llegar";
				}
				if (f() != "profundo") print 1 / 0;
			`,
		},
		{
			name: "corta un while en el medio",
			source: `
				var vueltas = 0;
				fun buscar(objetivo) {
					var i = 0;
					while (true) {
						vueltas = vueltas + 1;
						if (i == objetivo) return i;
						i = i + 1;
					}
				}
				if (buscar(4) != 4) print 1 / 0;
				if (vueltas != 5) print 1 / 0;
			`,
		},
		{
			name: "corta un for en el medio",
			source: `
				fun primerMultiplo(n) {
					for (var i = 1; i < 100; i = i + 1) {
						if (i % n == 0) return i;
					}
					return nil;
				}
				if (primerMultiplo(7) != 7) print 1 / 0;
			`,
		},
		{
			name: "el entorno de quien llama se restaura tras el return",
			source: `
				var x = "global";
				fun f() {
					var x = "local";
					{ var x = "bloque"; return x; }
				}
				var r = f();
				if (r != "bloque") print 1 / 0;
				if (x != "global") print 1 / 0;
			`,
		},
		{
			name: "devolver una closure (make_counter de la cátedra)",
			source: `
				fun makeCounter() {
					var i = 0;
					fun count() {
						i = i + 1;
						return i;
					}
					return count;
				}
				var a = makeCounter();
				var b = makeCounter();
				if (a() != 1) print 1 / 0;
				if (a() != 2) print 1 / 0;
				if (b() != 1) print 1 / 0;
				if (a() != 3) print 1 / 0;
			`,
		},
		{
			name: "el return de la función interna no corta la externa",
			source: `
				fun externa() {
					fun interna() { return 1; }
					var r = interna();
					return r + 1;
				}
				if (externa() != 2) print 1 / 0;
			`,
		},
		{
			name: "el valor del return se evalúa una sola vez",
			source: `
				var llamadas = 0;
				fun contar() { llamadas = llamadas + 1; return llamadas; }
				fun f() { return contar(); }
				f();
				if (llamadas != 1) print 1 / 0;
			`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := interpretSource(t, c.source); err != nil {
				t.Fatalf("interpretSource falló: %v", err)
			}
		})
	}
}

func TestInterpretReturnErrors(t *testing.T) {
	cases := []struct {
		name    string
		source  string
		wantMsg string
	}{
		// Un `return` fuera de una función es un error estático: lo prueba el
		// paquete resolver, porque ni siquiera llega a ejecutarse.
		{"error al evaluar el valor del return", "fun f() { return 1 / 0; } f();", "división por cero"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := interpretSource(t, c.source)
			if err == nil {
				t.Fatalf("esperaba un error de runtime en %q", c.source)
			}
			if !strings.Contains(err.Error(), c.wantMsg) {
				t.Fatalf("error = %q, esperaba que contenga %q", err, c.wantMsg)
			}
		})
	}
}

func TestStringifyFunction(t *testing.T) {
	fn := &interpreter.Function{
		Declaration: &ast.FunDecl{Name: token.Token{Type: token.IDENTIFIER, Lexeme: "suma"}},
	}
	if got := interpreter.Stringify(fn); got != "<fn suma>" {
		t.Fatalf("Stringify(función) = %q, esperaba %q", got, "<fn suma>")
	}
}
