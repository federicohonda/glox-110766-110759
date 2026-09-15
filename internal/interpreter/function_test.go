package interpreter_test

import (
	"strings"
	"testing"

	"github.com/federicohonda/glox-110766-110759/internal/ast"
	"github.com/federicohonda/glox-110766-110759/internal/interpreter"
	"github.com/federicohonda/glox-110766-110759/internal/token"
)

// Todavía no existe `return`, así que los resultados se sacan de las
// funciones asignando a variables de afuera. Igual que en el resto de los
// tests, si algo no da lo esperado se fuerza un error con `print 1 / 0`.
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

func TestStringifyFunction(t *testing.T) {
	fn := &interpreter.Function{
		Declaration: &ast.FunDecl{Name: token.Token{Type: token.IDENTIFIER, Lexeme: "suma"}},
	}
	if got := interpreter.Stringify(fn); got != "<fn suma>" {
		t.Fatalf("Stringify(función) = %q, esperaba %q", got, "<fn suma>")
	}
}
