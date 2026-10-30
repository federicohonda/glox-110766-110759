package bytecode_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/federicohonda/glox-110766-110759/internal/bytecode"
)

// Cada función queda compilada a su propio chunk, guardado como constante del
// chunk que la declara, y el disassembler los lista a continuación. Es el
// make_counter de las pruebas de la cátedra: `i` se captura, así que se
// envuelve en una celda al declararse; `count` es una local común.
func TestCompileFuncionACadaUnaSuChunk(t *testing.T) {
	chunk := mustCompileProgram(t, `fun make_counter() {
  var i = 0;
  fun count() {
    i = i + 1;
    return i;
  }
  return count;
}
print make_counter()();
`)
	expected := strings.Join([]string{
		"== programa ==",
		"0000    1 OP_CLOSURE          0 '<fn make_counter>'",
		"0002    | OP_DEFINE_GLOBAL    1 'make_counter'",
		"0004    9 OP_GET_GLOBAL       1 'make_counter'",
		"0006    | OP_CALL             0",
		"0008    | OP_CALL             0",
		"0010    | OP_PRINT",
		"0011    | OP_RETURN",
		"== <fn make_counter> ==",
		"0000    2 OP_CONSTANT         0 '0'",
		"0002    | OP_BOX              1",
		"0004    3 OP_CLOSURE          1 '<fn count>'",
		"     |                     local 1",
		"0008    7 OP_GET_LOCAL        2",
		"0010    | OP_RETURN",
		"0011    | OP_CONSTANT         2 'nil'",
		"0013    | OP_RETURN",
		"== <fn count> ==",
		"0000    4 OP_GET_CAPTURED     0",
		"0002    | OP_CONSTANT         0 '1'",
		"0004    | OP_ADD",
		"0005    | OP_SET_CAPTURED     0",
		"0007    | OP_POP",
		"0008    5 OP_GET_CAPTURED     0",
		"0010    | OP_RETURN",
		"0011    | OP_CONSTANT         1 'nil'",
		"0013    | OP_RETURN",
		"",
	}, "\n")
	if got := chunk.Disassemble("programa"); got != expected {
		t.Errorf("Disassemble distinto\n--- obtenido\n%s--- esperado\n%s", got, expected)
	}

	fn := chunk.Constants[0].(*bytecode.Function)
	if fn.Name != "make_counter" || fn.Arity != 0 || fn.CapturedCount != 0 {
		t.Errorf("make_counter = %+v", fn)
	}
	count := fn.Chunk.Constants[1].(*bytecode.Function)
	if count.CapturedCount != 1 {
		t.Errorf("count captura %d variables, se esperaba 1", count.CapturedCount)
	}
}

// Las locales que nadie captura no pagan la celda: fib usa slots comunes.
func TestCompileSinCapturasNoHayCeldas(t *testing.T) {
	chunk := mustCompileProgram(t, "fun fib(n) { if (n < 2) return n; return fib(n - 2) + fib(n - 1); }")
	listing := chunk.Disassemble("fib")
	for _, op := range []string{"OP_BOX", "OP_GET_CELL", "OP_SET_CELL", "OP_GET_CAPTURED"} {
		if strings.Contains(listing, op) {
			t.Errorf("el listado tiene %s y nada se captura:\n%s", op, listing)
		}
	}
}

// Los casos que el diseño marcó como "a no olvidar".
func TestCompileCasosDeCaptura(t *testing.T) {
	tests := []struct {
		name, source string
		// fragments tienen que aparecer, en orden, en el listado.
		fragments []string
	}{
		{
			name:   "parámetro capturado: el prólogo lo envuelve",
			source: "fun f(n) { fun g() { return n; } return g; }",
			fragments: []string{
				"== <fn f> ==", "OP_BOX              1", "OP_CLOSURE", "local 1",
			},
		},
		{
			name:   "función local recursiva: celda antes que closure",
			source: "{ fun f() { f(); } }",
			fragments: []string{
				"OP_CONSTANT", "'nil'", "OP_BOX              0", "OP_CLOSURE", "local 0", "OP_SET_CELL         0", "OP_POP",
				"== <fn f> ==", "OP_GET_CAPTURED     0", "OP_CALL             0",
			},
		},
		{
			name:   "tres niveles: la función del medio también captura",
			source: "fun a() { var x = 1; fun b() { fun c() { return x; } return c; } return b; }",
			fragments: []string{
				"== <fn a> ==", "OP_BOX              1", "local 1",
				"== <fn b> ==", "OP_CLOSURE", "captured 0",
				"== <fn c> ==", "OP_GET_CAPTURED     0",
			},
		},
		{
			name:   "bug de closures: se resuelve a la global y queda fija",
			source: `var a = "global"; { fun showA() { print a; } var a = "block"; }`,
			fragments: []string{
				"== <fn showA> ==", "OP_GET_GLOBAL",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listing := mustCompileProgram(t, tt.source).Disassemble("prueba")
			rest := listing
			for _, fragment := range tt.fragments {
				i := strings.Index(rest, fragment)
				if i == -1 {
					t.Fatalf("falta %q (en orden) en el listado:\n%s", fragment, listing)
				}
				rest = rest[i+len(fragment):]
			}
		})
	}
}

// Una función no puede tener más de 255 locales propias, pero sí capturar de
// varios niveles: c usa 200 variables de a y 100 de b. La que no entra es
// la 257, b56, en la línea 560.
func TestCompileLimiteDeCapturas(t *testing.T) {
	source := func(deB int) string {
		var sb strings.Builder
		sb.WriteString("fun a() {\n")
		for i := range 200 {
			fmt.Fprintf(&sb, "var a%d;\n", i)
		}
		sb.WriteString("fun b() {\n")
		for i := range deB {
			fmt.Fprintf(&sb, "var b%d;\n", i)
		}
		sb.WriteString("fun c() {\n")
		for i := range 200 {
			fmt.Fprintf(&sb, "print a%d;\n", i)
		}
		for i := range deB {
			fmt.Fprintf(&sb, "print b%d;\n", i)
		}
		sb.WriteString("}\n}\n}\n")
		return sb.String()
	}

	if _, err := compileProgramSource(t, source(56)); err != nil {
		t.Errorf("256 capturas deberían entrar: %v", err)
	}
	_, err := compileProgramSource(t, source(100))
	expected := "[línea 560] Error de compilación: demasiadas variables capturadas en una función (el máximo es 256)."
	if err == nil || err.Error() != expected {
		t.Errorf("error = %v\nesperado  %s", err, expected)
	}
}

func TestVMFunciones(t *testing.T) {
	tests := []struct {
		name, source, expected string
	}{
		{"llamada simple con argumentos", "fun suma(a, b) { return a + b; } print suma(1, 2);", "3\n"},
		{"sin return devuelve nil", "fun f() {} print f();", "nil\n"},
		{"return vacío", "fun f() { return; print 1; } print f();", "nil\n"},
		{"return corta en el medio de un loop", "fun f() { while (true) { return 7; } } print f();", "7\n"},
		{"se imprimen como <fn nombre>", "fun f() {} print f;", "<fn f>\n"},
		{"los argumentos se evalúan en orden", "fun p(x) { print x; return x; } fun f(a, b, c) {} f(p(1), p(2), p(3));", "1\n2\n3\n"},
		{"locales del llamador no se pisan", "fun f(a) { var b = a * 2; return b; } { var x = 1; var y = f(10); print x; print y; }", "1\n20\n"},
		{"llamadas anidadas", "fun f(a) { return a + 1; } print f(f(f(1)));", "4\n"},
		{"recursión: fib", "fun fib(n) { if (n < 2) return n; return fib(n - 2) + fib(n - 1); } print fib(20);", "6765\n"},
		{"recursión mutua", "fun par(n) { if (n == 0) return true; return impar(n - 1); } fun impar(n) { if (n == 0) return false; return par(n - 1); } print par(10); print impar(7);", "true\ntrue\n"},
		{"funciones de orden superior", "fun dos(f, x) { return f(f(x)); } fun inc(x) { return x + 1; } print dos(inc, 5);", "7\n"},
		{"la función es un valor", "fun f() { return 1; } var g = f; print g(); print f == g;", "1\ntrue\n"},
		{"cada declaración es una función distinta", "fun mk() { fun f() {} return f; } print mk() == mk();", "false\n"},
		{"recursión profunda", "fun s(n) { if (n == 0) return 0; return n + s(n - 1); } print s(10000);", "5.0005e+07\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mustRunProgram(t, tt.source); got != tt.expected {
				t.Errorf("salida = %q, esperado %q", got, tt.expected)
			}
		})
	}
}

func TestVMClosures(t *testing.T) {
	tests := []struct {
		name, source, expected string
	}{
		{
			"make_counter con dos contadores independientes",
			"fun mc() { var i = 0; fun c() { i = i + 1; return i; } return c; } var a = mc(); var b = mc(); print a(); print a(); print b(); print a();",
			"1\n2\n1\n3\n",
		},
		{
			"tres niveles",
			`fun a() { var x = "x"; fun b() { fun c() { x = x + "!"; return x; } return c; } return b; } var f = a()(); print f(); print f();`,
			"x!\nx!!\n",
		},
		{
			"bug de closures",
			`var a = "global"; { fun showA() { print a; } showA(); var a = "block"; showA(); }`,
			"global\nglobal\n",
		},
		{
			"dos closures comparten la misma variable",
			"fun par() { var n = 0; fun inc() { n = n + 1; } fun get() { return n; } inc(); inc(); return get; } print par()();",
			"2\n",
		},
		{
			"la función que declara ve lo que cambia la interna",
			"fun f() { var n = 1; fun g() { n = 10; } g(); return n; } print f();",
			"10\n",
		},
		{
			"parámetro capturado",
			"fun sumador(n) { fun s(m) { return n + m; } return s; } print sumador(3)(4);",
			"7\n",
		},
		{
			"función local recursiva",
			"{ fun fact(n) { if (n <= 1) return 1; return n * fact(n - 1); } print fact(5); }",
			"120\n",
		},
		{
			"cada vuelta del loop tiene su propia celda",
			"var f0; var f1; for (var i = 0; i < 2; i = i + 1) { var j = i; fun g() { return j; } if (i == 0) f0 = g; else f1 = g; } print f0(); print f1();",
			"0\n1\n",
		},
		{
			"la variable del for es una sola para todo el loop",
			"var f; for (var i = 0; i < 3; i = i + 1) { fun g() { return i; } if (i == 0) f = g; } print f();",
			"3\n",
		},
		{
			"la closure sobrevive al bloque",
			`var g; { var local = "capturada"; fun leer() { return local; } g = leer; } print g();`,
			"capturada\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mustRunProgram(t, tt.source); got != tt.expected {
				t.Errorf("salida = %q, esperado %q", got, tt.expected)
			}
		})
	}
}

func TestVMErroresDeLlamada(t *testing.T) {
	tests := []struct {
		name, source, expected string
	}{
		{"llamar a algo que no es función", `"hola"();`, "[línea 1] Error en tiempo de ejecución en ')': solo se pueden llamar funciones, se obtuvo: hola"},
		{"llamar a nil", "var f; f();", "[línea 1] Error en tiempo de ejecución en ')': solo se pueden llamar funciones, se obtuvo: nil"},
		{"aridad", "fun f(a, b) {}\nf(1);", "[línea 2] Error en tiempo de ejecución en ')': se esperaban 2 argumentos pero se recibieron 1."},
		{"error dentro de una función: su línea", "fun f() {\n  return 1 / 0;\n}\nf();", "[línea 2] Error en tiempo de ejecución en '/': división por cero."},
		{"recursión infinita", "fun f() { f(); } f();", "[línea 1] Error en tiempo de ejecución en ')': desbordamiento de pila (más de 100000 llamadas anidadas)."},
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

// Un error en el medio de una llamada deja frames y pila a medio armar: la
// próxima corrida (la próxima línea del REPL) tiene que arrancar limpia.
func TestVMSeReutilizaDespuesDeUnErrorEnUnaFuncion(t *testing.T) {
	var out strings.Builder
	vm := bytecode.NewVM(&out)
	if _, err := vm.Run(mustCompileProgram(t, "fun f(n) { var x = 1; return n / 0; }")); err != nil {
		t.Fatal(err)
	}
	if _, err := vm.Run(mustCompileProgram(t, "print f(1);")); err == nil {
		t.Fatal("se esperaba división por cero")
	}
	if _, err := vm.Run(mustCompileProgram(t, "{ var a = 5; print a; }")); err != nil {
		t.Fatal(err)
	}
	if out.String() != "5\n" {
		t.Errorf("salida = %q", out.String())
	}
}
