package interpreter_test

import (
	"testing"

	"github.com/federicohonda/glox-110766-110759/internal/interpreter"
	"github.com/federicohonda/glox-110766-110759/internal/token"
)

func makeToken(lexeme string) token.Token {
	return token.Token{
		Type:    token.IDENTIFIER,
		Lexeme:  lexeme,
		Literal: nil,
		Line:    1,
	}
}

func TestEnvironmentDefineAndGet(t *testing.T) {
	env := interpreter.NewEnvironment()
	tokA := makeToken("a")

	env.Define("a", 42.0)

	val, err := env.Get(tokA)
	if err != nil {
		t.Fatalf("se esperaba encontrar 'a', pero ocurrió un error: %v", err)
	}
	if val != 42.0 {
		t.Errorf("Get('a') = %v, esperado 42.0", val)
	}
}

func TestEnvironmentGetUndefined(t *testing.T) {
	env := interpreter.NewEnvironment()
	tokX := makeToken("x")

	_, err := env.Get(tokX)
	if err == nil {
		t.Fatal("se esperaba error al buscar variable no definida")
	}

	if _, ok := err.(*interpreter.RuntimeError); !ok {
		t.Errorf("se esperaba *interpreter.RuntimeError, se obtuvo %T", err)
	}
}

func TestEnvironmentAssign(t *testing.T) {
	env := interpreter.NewEnvironment()
	tokA := makeToken("a")

	env.Define("a", 10.0)

	err := env.Assign(tokA, 20.0)
	if err != nil {
		t.Fatalf("error inesperado al reasignar 'a': %v", err)
	}

	val, err := env.Get(tokA)
	if err != nil {
		t.Fatalf("error inesperado al leer 'a': %v", err)
	}
	if val != 20.0 {
		t.Errorf("después de reasignar, Get('a') = %v, esperado 20.0", val)
	}
}

func TestEnvironmentAssignUndefined(t *testing.T) {
	env := interpreter.NewEnvironment()
	tokX := makeToken("x")

	err := env.Assign(tokX, 100.0)
	if err == nil {
		t.Fatal("se esperaba error al reasignar variable no definida")
	}

	if _, ok := err.(*interpreter.RuntimeError); !ok {
		t.Errorf("se esperaba *interpreter.RuntimeError, se obtuvo %T", err)
	}
}

func TestEnvironmentNestingAndShadowing(t *testing.T) {
	global := interpreter.NewEnvironment()
	tokGlobal := makeToken("global")
	tokShadow := makeToken("x")

	global.Define("global", "valor_global")
	global.Define("x", 1.0)

	// Crear entorno hijo (bloque interno)
	inner := interpreter.NewEnclosingEnvironment(global)

	// El entorno hijo puede leer del entorno padre
	valGlobal, err := inner.Get(tokGlobal)
	if err != nil || valGlobal != "valor_global" {
		t.Errorf("inner.Get('global') = %v (err: %v), esperado 'valor_global'", valGlobal, err)
	}

	// El entorno hijo define 'x', sombreando (shadowing) a 'x' del padre
	inner.Define("x", 2.0)

	valInnerX, err := inner.Get(tokShadow)
	if err != nil || valInnerX != 2.0 {
		t.Errorf("inner.Get('x') = %v, esperado 2.0", valInnerX)
	}

	// El entorno padre mantiene su propio 'x' intacto
	valGlobalX, err := global.Get(tokShadow)
	if err != nil || valGlobalX != 1.0 {
		t.Errorf("global.Get('x') = %v, esperado 1.0", valGlobalX)
	}
}

func TestEnvironmentNestedAssignment(t *testing.T) {
	// Reasignar una variable del padre desde el hijo modifica el valor en el padre
	parent := interpreter.NewEnvironment()
	tokX := makeToken("x")
	parent.Define("x", 10.0)

	child := interpreter.NewEnclosingEnvironment(parent)

	err := child.Assign(tokX, 99.0)
	if err != nil {
		t.Fatalf("error inesperado al reasignar en child: %v", err)
	}

	valParent, err := parent.Get(tokX)
	if err != nil || valParent != 99.0 {
		t.Errorf("parent.Get('x') tras reasignar en child = %v, esperado 99.0", valParent)
	}
}
