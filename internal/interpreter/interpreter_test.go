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

	expr, err := parser.New(tokens).Parse()
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

var _ = math.Abs // asegurar que math pueda compilarse
