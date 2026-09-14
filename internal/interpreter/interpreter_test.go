package interpreter_test

import (
	"math"
	"testing"

	"github.com/federicohonda/glox-110766-110759/internal/ast"
	"github.com/federicohonda/glox-110766-110759/internal/interpreter"
	"github.com/federicohonda/glox-110766-110759/internal/parser"
	"github.com/federicohonda/glox-110766-110759/internal/scanner"
	"github.com/federicohonda/glox-110766-110759/internal/token"
)

func evalSource(t *testing.T, source string) (interpreter.Value, error) {
	t.Helper()
	tokens, errs := scanner.New(source).Scan()
	if len(errs) > 0 {
		t.Fatalf("error léxico inesperado al escanear %q: %v", source, errs)
	}

	expr, err := parser.New(tokens).ParseExpression()
	if err != nil {
		t.Fatalf("error sintáctico inesperado al parsear %q: %v", source, err)
	}

	interp := interpreter.New()
	return interp.Evaluate(expr)
}

func mustEval(t *testing.T, source string) interpreter.Value {
	t.Helper()
	val, err := evalSource(t, source)
	if err != nil {
		t.Fatalf("evalSource(%q) falló con error: %v", source, err)
	}
	return val
}

func TestEvaluateLiterals(t *testing.T) {
	tests := []struct {
		source   string
		expected any
	}{
		{"123", 123.0},
		{"98.76", 98.76},
		{"\"hola\"", "hola"},
		{"true", true},
		{"false", false},
		{"nil", nil},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			got := mustEval(t, tt.source)
			if got != tt.expected {
				t.Errorf("evalSource(%q) = %v (%T), esperado %v (%T)", tt.source, got, got, tt.expected, tt.expected)
			}
		})
	}
}

func TestEvaluateGrouping(t *testing.T) {
	got := mustEval(t, "(42)")
	if got != 42.0 {
		t.Errorf("evalSource(\"(42)\") = %v, esperado 42", got)
	}
}

func TestEvaluateUnary(t *testing.T) {
	tests := []struct {
		source   string
		expected any
	}{
		{"-5", -5.0},
		{"--5", 5.0},
		{"!true", false},
		{"!false", true},
		{"!nil", true},
		{"!0", false},        // 0 es truthy en Lox
		{"!\"\"", false},     // string vacío es truthy en Lox
		{"!\"hola\"", false}, // strings no vacíos son truthy
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			got := mustEval(t, tt.source)
			if got != tt.expected {
				t.Errorf("evalSource(%q) = %v, esperado %v", tt.source, got, got)
			}
		})
	}
}

func TestEvaluateUnaryTypeErrors(t *testing.T) {
	invalidSources := []string{
		"-\"hola\"",
		"-true",
		"-nil",
	}

	for _, src := range invalidSources {
		t.Run(src, func(t *testing.T) {
			val, err := evalSource(t, src)
			if err == nil {
				t.Fatalf("evalSource(%q) debía fallar por error de tipo, pero devolvió %v", src, val)
			}
			if _, ok := err.(*interpreter.RuntimeError); !ok {
				t.Errorf("se esperaba *interpreter.RuntimeError, se obtuvo %T: %v", err, err)
			}
		})
	}
}

func TestEvaluateArithmetic(t *testing.T) {
	tests := []struct {
		source   string
		expected any
	}{
		{"1 + 2", 3.0},
		{"10 - 4", 6.0},
		{"3 * 4", 12.0},
		{"15 / 3", 5.0},
		{"10 % 3", 1.0},
		{"1 + 2 * 3", 7.0},   // precedencia de * sobre +
		{"(1 + 2) * 3", 9.0}, // paréntesis alteran precedencia
		{"10 - 2 - 3", 5.0},  // asociatividad a izquierda: (10 - 2) - 3 = 5
		{"20 / 4 / 2", 2.5},  // asociatividad a izquierda: (20 / 4) / 2 = 2.5
		{"\"hola \" + \"mundo\"", "hola mundo"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			got := mustEval(t, tt.source)
			if got != tt.expected {
				t.Errorf("evalSource(%q) = %v, esperado %v", tt.source, got, got)
			}
		})
	}
}

func TestEvaluateComparison(t *testing.T) {
	tests := []struct {
		source   string
		expected bool
	}{
		{"5 > 3", true},
		{"3 > 5", false},
		{"5 >= 5", true},
		{"4 >= 5", false},
		{"2 < 7", true},
		{"7 < 2", false},
		{"3 <= 3", true},
		{"4 <= 3", false},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			got := mustEval(t, tt.source)
			if got != tt.expected {
				t.Errorf("evalSource(%q) = %v, esperado %v", tt.source, got, got)
			}
		})
	}
}

func TestEvaluateEquality(t *testing.T) {
	tests := []struct {
		source   string
		expected bool
	}{
		{"nil == nil", true},
		{"nil != nil", false},
		{"nil == false", false},
		{"true == true", true},
		{"true == false", false},
		{"false == false", true},
		{"1 == 1", true},
		{"1 == 2", false},
		{"1 != 2", true},
		{"\"hola\" == \"hola\"", true},
		{"\"hola\" != \"mundo\"", true},
		// Sin coerción de tipos (a diferencia de JS):
		{"\"1\" == 1", false},
		{"\"true\" == true", false},
		{"0 == false", false},
		{"\"\" == false", false},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			got := mustEval(t, tt.source)
			if got != tt.expected {
				t.Errorf("evalSource(%q) = %v, esperado %v", tt.source, got, got)
			}
		})
	}
}

func TestEvaluateTypeErrors(t *testing.T) {
	tests := []string{
		"\"1\" + 1",
		"1 + \"1\"",
		"\"a\" - \"b\"",
		"\"a\" * 2",
		"\"a\" / 2",
		"\"a\" % 2",
		"\"a\" < \"b\"",
		"\"a\" <= \"b\"",
		"\"a\" > \"b\"",
		"\"a\" >= \"b\"",
		"true + 1",
		"nil + nil",
	}

	for _, src := range tests {
		t.Run(src, func(t *testing.T) {
			val, err := evalSource(t, src)
			if err == nil {
				t.Fatalf("evalSource(%q) debía fallar por error de tipo, pero devolvió %v", src, val)
			}
			if _, ok := err.(*interpreter.RuntimeError); !ok {
				t.Errorf("se esperaba *interpreter.RuntimeError, se obtuvo %T: %v", err, err)
			}
		})
	}
}

func TestEvaluateDivisionByZero(t *testing.T) {
	tests := []string{
		"10 / 0",
		"10 % 0",
	}

	for _, src := range tests {
		t.Run(src, func(t *testing.T) {
			val, err := evalSource(t, src)
			if err == nil {
				t.Fatalf("evalSource(%q) debía fallar por división/módulo por cero, pero devolvió %v", src, val)
			}
			if _, ok := err.(*interpreter.RuntimeError); !ok {
				t.Errorf("se esperaba *interpreter.RuntimeError, se obtuvo %T: %v", err, err)
			}
		})
	}
}

func TestEvaluationOrderGoldenRule(t *testing.T) {
	// Regla de oro de Lox: en una expresión binaria, se evalúan ambos operandos
	// antes de chequear tipos. Si ambos operandos son válidos como expresiones
	// pero incompatibles entre sí, la evaluación de ambos debe completarse antes
	// del error de tipo.
	tok := token.Token{Type: token.PLUS, Lexeme: "+", Line: 1}
	bin := &ast.Binary{
		Left:     &ast.Literal{Value: "texto"},
		Operator: tok,
		Right:    &ast.Literal{Value: 123.0},
	}

	interp := interpreter.New()
	_, err := interp.Evaluate(bin)
	if err == nil {
		t.Fatal("se esperaba error de tipos para 'texto' + 123")
	}

	rtErr, ok := err.(*interpreter.RuntimeError)
	if !ok {
		t.Fatalf("se esperaba *interpreter.RuntimeError, se obtuvo %T", err)
	}
	if rtErr.Token.Lexeme != "+" {
		t.Errorf("token esperado '+', se obtuvo %q", rtErr.Token.Lexeme)
	}
}

func TestStringify(t *testing.T) {
	tests := []struct {
		input    interpreter.Value
		expected string
	}{
		{nil, "nil"},
		{true, "true"},
		{false, "false"},
		{123.0, "123"},
		{12.34, "12.34"},
		{"hola", "hola"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := interpreter.Stringify(tt.input)
			if got != tt.expected {
				t.Errorf("Stringify(%v) = %q, esperado %q", tt.input, got, tt.expected)
			}
		})
	}
}

func interpretSource(t *testing.T, source string) (*interpreter.Interpreter, error) {
	t.Helper()
	tokens, errs := scanner.New(source).Scan()
	if len(errs) > 0 {
		t.Fatalf("error léxico inesperado al escanear %q: %v", source, errs)
	}

	stmts, parseErrs := parser.New(tokens).Parse()
	if len(parseErrs) > 0 {
		t.Fatalf("error sintáctico inesperado al parsear %q: %v", source, parseErrs)
	}

	interp := interpreter.New()
	err := interp.Interpret(stmts)
	return interp, err
}

func TestInterpretStatements(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{
			name:   "declaración y uso de variables",
			source: "var x = 10; var y = x + 5; print y;",
		},
		{
			name:   "declaración sin inicializador",
			source: "var a; print a;",
		},
		{
			name:   "reasignación de variable",
			source: "var a = 1; a = 2; print a;",
		},
		{
			name:   "asignación encadenada",
			source: "var a; var b; a = b = 42; print a; print b;",
		},
		{
			name:   "expression statements",
			source: "1 + 2; \"hola\"; true;",
		},
		{
			name:   "bloques de código",
			source: "{ var a = 1; print a; }",
		},
		{
			name:   "bloques anidados",
			source: "{ { var b = 2; print b; } }",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := interpretSource(t, c.source)
			if err != nil {
				t.Fatalf("interpretSource(%q) falló inesperadamente: %v", c.source, err)
			}
		})
	}
}

func TestInterpretUndefinedVariable(t *testing.T) {
	errorSources := []string{
		"print variableNoDefinida;",
		"variableNoDefinida = 42;",
		"var a = variableNoDefinida;",
	}

	for _, src := range errorSources {
		t.Run(src, func(t *testing.T) {
			_, err := interpretSource(t, src)
			if err == nil {
				t.Fatalf("se esperaba RuntimeError en %q, pero no hubo error", src)
			}
			if _, ok := err.(*interpreter.RuntimeError); !ok {
				t.Errorf("se esperaba *interpreter.RuntimeError, se obtuvo %T: %v", err, err)
			}
		})
	}
}

func TestInterpretLexicalScopingAndShadowing(t *testing.T) {
	// Checkpoint del plan: var x = 1; { var x = 2; print x; } print x;
	// Además verificamos que una variable local no se fugue al scope exterior
	cases := []struct {
		name   string
		source string
	}{
		{
			name: "checkpoint: shadowing en bloque",
			source: `
				var x = 1;
				{
					var x = 2;
					print x;
				}
				print x;
			`,
		},
		{
			name: "scoping anidado en tres niveles",
			source: `
				var a = "global a";
				var b = "global b";
				var c = "global c";
				{
					var a = "outer a";
					var b = "outer b";
					{
						var a = "inner a";
						print a;
						print b;
						print c;
					}
					print a;
					print b;
					print c;
				}
				print a;
				print b;
				print c;
			`,
		},
		{
			name: "reasignar variable exterior desde bloque interno",
			source: `
				var x = "inicial";
				{
					x = "modificado";
				}
				print x;
			`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := interpretSource(t, c.source)
			if err != nil {
				t.Fatalf("interpretSource(%q) falló inesperadamente: %v", c.source, err)
			}
		})
	}
}

func TestInterpretInnerVariableNotAccessibleOutside(t *testing.T) {
	source := `
		{
			var local = "secreto";
		}
		print local;
	`
	_, err := interpretSource(t, source)
	if err == nil {
		t.Fatal("se esperaba RuntimeError al intentar acceder a una variable de un bloque ya cerrado")
	}
	if _, ok := err.(*interpreter.RuntimeError); !ok {
		t.Errorf("se esperaba *interpreter.RuntimeError, se obtuvo %T: %v", err, err)
	}
}

func TestInterpretIfStatement(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{
			name:   "if con condición verdadera ejecuta then",
			source: "var a = 0; if (true) a = 1; print a;",
		},
		{
			name:   "if con condición falsa no ejecuta then",
			source: "var a = 0; if (false) a = 1; print a;",
		},
		{
			name:   "if-else con condición verdadera ejecuta then y no else",
			source: "var a = 0; if (true) a = 1; else a = 2; print a;",
		},
		{
			name:   "if-else con condición falsa ejecuta else y no then",
			source: "var a = 0; if (false) a = 1; else a = 2; print a;",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := interpretSource(t, c.source)
			if err != nil {
				t.Fatalf("interpretSource(%q) falló: %v", c.source, err)
			}
		})
	}
}

func TestInterpretLogicalExpressions(t *testing.T) {
	tests := []struct {
		source   string
		expected any
	}{
		{"nil or \"hola\"", "hola"},
		{"\"primero\" or \"segundo\"", "primero"},
		{"nil and \"hola\"", nil},
		{"\"primero\" and \"segundo\"", "segundo"},
		{"false and true", false},
		{"true and true", true},
		{"false or true", true},
		{"false or false", false},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			got := mustEval(t, tt.source)
			if got != tt.expected {
				t.Errorf("evalSource(%q) = %v, esperado %v", tt.source, got, tt.expected)
			}
		})
	}
}

func TestInterpretShortCircuiting(t *testing.T) {
	// Checkpoint del plan: un and/or con efectos del lado que no debería evaluarse
	// no debe ejecutarse (cortocircuito).
	cases := []struct {
		name   string
		source string
	}{
		{
			name: "or cortocircuita cuando izquierdo es verdadero",
			source: `
				var ejecutado = false;
				true or (ejecutado = true);
				if (ejecutado) print 1 / 0; // si evaluó el lado derecho, falla
			`,
		},
		{
			name: "and cortocircuita cuando izquierdo es falso",
			source: `
				var ejecutado = false;
				false and (ejecutado = true);
				if (ejecutado) print 1 / 0; // si evaluó el lado derecho, falla
			`,
		},
		{
			name: "or evalúa derecho cuando izquierdo es falso",
			source: `
				var ejecutado = false;
				false or (ejecutado = true);
				if (!ejecutado) print 1 / 0;
			`,
		},
		{
			name: "and evalúa derecho cuando izquierdo es verdadero",
			source: `
				var ejecutado = false;
				true and (ejecutado = true);
				if (!ejecutado) print 1 / 0;
			`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := interpretSource(t, c.source)
			if err != nil {
				t.Fatalf("interpretSource(%q) falló por cortocircuito erróneo: %v", c.source, err)
			}
		})
	}
}

func TestInterpretWhileLoop(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{
			name: "while cuenta sumatoria de 1 a 5",
			source: `
				var i = 1;
				var suma = 0;
				while (i <= 5) {
					suma = suma + i;
					i = i + 1;
				}
				if (suma != 15) print 1 / 0;
			`,
		},
		{
			name: "while con condición falsa no ejecuta cuerpo",
			source: `
				var ejecutado = false;
				while (false) {
					ejecutado = true;
				}
				if (ejecutado) print 1 / 0;
			`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := interpretSource(t, c.source)
			if err != nil {
				t.Fatalf("interpretSource(%q) falló: %v", c.source, err)
			}
		})
	}
}

func TestInterpretForLoop(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{
			name: "for clásico con sumatoria",
			source: `
				var suma = 0;
				for (var i = 0; i < 10; i = i + 1) {
					suma = suma + i;
				}
				if (suma != 45) print 1 / 0;
			`,
		},
		{
			name: "for sin inicializador",
			source: `
				var suma = 0;
				var i = 0;
				for (; i < 5; i = i + 1) {
					suma = suma + 1;
				}
				if (suma != 5) print 1 / 0;
			`,
		},
		{
			name: "for sin incremento",
			source: `
				var count = 0;
				for (var i = 0; i < 3;) {
					count = count + 1;
					i = i + 1;
				}
				if (count != 3) print 1 / 0;
			`,
		},
		{
			name: "for anidados",
			source: `
				var total = 0;
				for (var x = 0; x < 3; x = x + 1) {
					for (var y = 0; y < 3; y = y + 1) {
						total = total + 1;
					}
				}
				if (total != 9) print 1 / 0;
			`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := interpretSource(t, c.source)
			if err != nil {
				t.Fatalf("interpretSource(%q) falló: %v", c.source, err)
			}
		})
	}
}

func TestInterpretForVariableScoping(t *testing.T) {
	// La variable declarada en el for debe estar encerrada en su bloque
	// y no existir fuera de él
	source := `
		for (var variableFor = 0; variableFor < 1; variableFor = variableFor + 1) {
			print variableFor;
		}
		print variableFor;
	`
	_, err := interpretSource(t, source)
	if err == nil {
		t.Fatal("se esperaba RuntimeError al acceder a la variable del for fuera del bucle")
	}
	if _, ok := err.(*interpreter.RuntimeError); !ok {
		t.Errorf("se esperaba *interpreter.RuntimeError, se obtuvo %T: %v", err, err)
	}
}

var _ = math.Abs // asegurar que math pueda compilarse
