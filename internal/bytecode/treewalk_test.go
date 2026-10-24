package bytecode_test

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/federicohonda/glox-110766-110759/internal/bytecode"
	"github.com/federicohonda/glox-110766-110759/internal/interpreter"
	"github.com/federicohonda/glox-110766-110759/internal/parser"
	"github.com/federicohonda/glox-110766-110759/internal/scanner"
)

// Estos tests usan al intérprete tree-walk como oráculo: la misma expresión,
// parseada una sola vez, se evalúa recorriendo el AST y también compilada a
// bytecode en la VM. Las dos tienen que dar el mismo valor o el mismo error,
// con el mismo mensaje y la misma línea. Como las reglas de cada operador
// están escritas dos veces (interpreter.go y vm.go), esto es lo que detecta
// que una de las dos se desvíe.

// sameValue compara dos valores de Lox. Los números se comparan bit a bit
// para distinguir 0 de -0, salvo NaN, que no es igual a sí mismo con ==.
func sameValue(a, b bytecode.Value) bool {
	fa, aIsNumber := a.(float64)
	fb, bIsNumber := b.(float64)
	if aIsNumber && bIsNumber {
		if math.IsNaN(fa) || math.IsNaN(fb) {
			return math.IsNaN(fa) && math.IsNaN(fb)
		}
		return math.Float64bits(fa) == math.Float64bits(fb)
	}
	return a == b
}

// outcome describe en una línea cómo terminó una evaluación.
func outcome(value bytecode.Value, err error) string {
	if err != nil {
		return "error: " + err.Error()
	}
	return fmt.Sprintf("valor: %v (%T)", value, value)
}

// checkSameAsTreeWalk evalúa la expresión con los dos backends, falla si no
// coinciden y devuelve si el resultado fue un error de runtime.
func checkSameAsTreeWalk(t *testing.T, source string) (failed bool) {
	t.Helper()
	tokens, errs := scanner.New(source).Scan()
	if len(errs) > 0 {
		t.Fatalf("error léxico inesperado al escanear %q: %v", source, errs)
	}
	expr, err := parser.New(tokens).ParseExpression()
	if err != nil {
		t.Fatalf("error sintáctico inesperado al parsear %q: %v", source, err)
	}

	wantValue, wantErr := interpreter.New().Evaluate(expr)

	chunk, err := bytecode.Compile(expr)
	if err != nil {
		t.Fatalf("Compile(%q) falló con error: %v", source, err)
	}
	gotValue, gotErr := bytecode.NewVM().Run(chunk)

	same := (wantErr == nil) == (gotErr == nil)
	if same && wantErr != nil {
		same = wantErr.Error() == gotErr.Error()
	}
	if same && wantErr == nil {
		same = sameValue(wantValue, gotValue)
	}
	if !same {
		t.Errorf("los dos backends no coinciden en %q\n tree-walk: %s\n bytecode:  %s",
			source, outcome(wantValue, wantErr), outcome(gotValue, gotErr))
	}
	return wantErr != nil
}

func TestVMCoincideConElTreeWalk(t *testing.T) {
	sources := []string{
		// Aritmética y precedencia.
		"1 + 2 * 3 - 4 / 5 % 6",
		"(1 + 2) * (3 - 4) / (5 % 6)",
		"-(-(-1))",
		"2 * -3",
		"10 % 3 % 2",
		"0.1 + 0.2",
		"1 / 3",
		"-0",
		"0 * -1",

		// Módulo con signos y decimales.
		"5 % 3", "-5 % 3", "5 % -3", "-5 % -3", "5.5 % 2", "0 % 5",

		// Strings y demás literales.
		`"a" + "b"`, `"" + ""`, `"1" + "2"`, `"línea 1
línea 2" + "!"`,
		"true", "false", "nil", `"hola"`,

		// División y módulo por cero, en todas sus formas.
		"1 / 0", "0 / 0", "1 / (2 - 2)", "1 / -0", "1 / (0 * 5)",
		"1 % 0", "0 % 0", "1 % (3 - 3)",

		// Tipos equivocados, de cada lado y en cada operador.
		`1 + "a"`, `"a" + 1`, "1 + nil", "true + false", `"a" + nil`,
		`"a" - 1`, `1 - "a"`, "nil - nil",
		`"a" * 2`, "2 * true",
		`"a" / 2`, `2 / "a"`, "nil / 0",
		`"a" % 2`, `2 % "a"`, "true % 0",
		`-"a"`, "-nil", "-true", `--"a"`,

		// Regla de oro: gana el error que aparece primero al evaluar, y los
		// tipos se miran antes que el divisor.
		`"a" * (1 / 0)`,
		`(1 / 0) + ("a" - 1)`,
		`("a" - 1) + (1 / 0)`,
		`-"a" + (1 / 0)`,
		`"a" / 0`,
		`0 % "a"`,

		// El error se reporta en la línea del operador que falló.
		"1 +\n2 +\n(3 / 0)",
		"1\n+\n\"a\"",
		"-\n\n\"a\"",
	}

	for _, source := range sources {
		t.Run(source, func(t *testing.T) {
			checkSameAsTreeWalk(t, source)
		})
	}
}

// Números tan grandes que la multiplicación se va de rango: en los dos
// backends da infinito (y de ahí NaN), no un error.
func TestVMCoincideConElTreeWalkFueraDeRango(t *testing.T) {
	huge := "1" + strings.Repeat("0", 200)
	sources := []string{
		huge + " * " + huge,
		"-" + huge + " * " + huge,
		huge + " * " + huge + " - " + huge + " * " + huge,
		huge + " * " + huge + " % 2",
		"1 / (" + huge + " * " + huge + ")",
	}

	for i, source := range sources {
		t.Run(fmt.Sprintf("caso %d", i), func(t *testing.T) {
			if failed := checkSameAsTreeWalk(t, source); failed {
				t.Errorf("no se esperaba un error de runtime")
			}
		})
	}
}

// exprGenerator arma expresiones al azar a partir de una semilla fija, así
// el test es reproducible.
type exprGenerator struct {
	rng *rand.Rand
}

var (
	numberLiterals = []string{"0", "1", "2", "3", "7", "10", "100", "0.5", "2.5", "0.125"}
	otherLiterals  = []string{`"a"`, `""`, `"lox"`, "nil", "true", "false"}
	binarySymbols  = []string{"+", "-", "*", "/", "%", "<", "<=", ">", ">=", "==", "!=", "and", "or"}
)

func (g *exprGenerator) pick(options []string) string {
	return options[g.rng.IntN(len(options))]
}

// space separa dos tokens con un espacio o, cada tanto, con un salto de
// línea, para que también se compare la línea de los errores.
func (g *exprGenerator) space() string {
	if g.rng.IntN(8) == 0 {
		return "\n"
	}
	return " "
}

// expr genera una expresión de a lo sumo `depth` niveles de anidamiento.
// otherChance es la probabilidad (en porcentaje) de que una hoja no sea un
// número.
func (g *exprGenerator) expr(depth, otherChance int) string {
	if depth == 0 || g.rng.IntN(4) == 0 {
		if g.rng.IntN(100) < otherChance {
			return g.pick(otherLiterals)
		}
		return g.pick(numberLiterals)
	}

	switch n := g.rng.IntN(10); {
	case n < 2:
		// El espacio evita que dos '-' seguidos queden pegados sin querer; al
		// scanner le da lo mismo, pero se lee mejor en el mensaje de falla.
		op := "-"
		if g.rng.IntN(2) == 0 {
			op = "!"
		}
		return op + g.space() + g.expr(depth-1, otherChance)
	case n < 4:
		return "(" + g.expr(depth-1, otherChance) + ")"
	default:
		return g.expr(depth-1, otherChance) + g.space() + g.pick(binarySymbols) + g.space() + g.expr(depth-1, otherChance)
	}
}

func TestVMCoincideConElTreeWalkEnExpresionesAlAzar(t *testing.T) {
	const total = 3000
	gen := &exprGenerator{rng: rand.New(rand.NewPCG(110766, 110759))}

	values, failures := 0, 0
	for i := range total {
		// Dos de cada tres expresiones son solo numéricas, para que una buena
		// parte llegue a un valor; el resto mezcla tipos y termina casi
		// siempre en error.
		otherChance := 0
		if i%3 == 0 {
			otherChance = 15
		}
		if checkSameAsTreeWalk(t, gen.expr(5, otherChance)) {
			failures++
		} else {
			values++
		}
		if t.Failed() {
			// Con la primera diferencia alcanza: no hace falta llenar la
			// salida con las que siguen.
			return
		}
	}

	// Si el generador degenerara en solo valores o solo errores, el test
	// seguiría pasando sin estar probando nada.
	if values < total/10 || failures < total/10 {
		t.Errorf("el corpus quedó desbalanceado: %d valores y %d errores de %d", values, failures, total)
	}
	t.Logf("%d expresiones: %d dieron un valor y %d un error de runtime", total, values, failures)
}
