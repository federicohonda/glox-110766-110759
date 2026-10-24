// Package tests corre glox como lo usaría cualquiera: compila el binario y lo
// ejecuta sobre scripts .lox, comparando stdout, stderr y código de salida.
//
// Cada script de tests/lox declara lo que espera con comentarios:
//
//	print 1 + 2; // expect: 3      una línea de stdout, en orden
//	// exit: 65                    código de salida (0 si no se indica)
//	// stderr: variable no definida   un fragmento que tiene que aparecer en stderr
//
// Además se verifica que los programas de examples/ produzcan exactamente la
// salida guardada en tests/golden/, y que los scripts de benchmarks/ den el
// resultado que declaran.
//
// Los scripts de tests/bytecode se corren con --disassemble. Cada uno es una
// expresión suelta y usa los mismos comentarios, más uno propio:
//
//	// ops: OP_CONSTANT OP_NEGATE OP_RETURN   las instrucciones del listado, en orden
//
// En ellos `// expect:` es el resultado que imprime la VM.
package tests

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "regenerar los archivos de tests/golden con la salida actual")

var gloxBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "glox-e2e")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	gloxBin = filepath.Join(dir, "glox")
	build := exec.Command("go", "build", "-o", gloxBin, "../cmd/glox")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "no se pudo compilar glox:", err)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

type result struct {
	stdout, stderr string
	exit           int
}

func runGlox(t *testing.T, args ...string) result {
	t.Helper()
	cmd := exec.Command(gloxBin, args...)
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	err := cmd.Run()
	exit := 0
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exit = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("no se pudo ejecutar glox: %v", err)
	}
	return result{out.String(), errOut.String(), exit}
}

var (
	expectRe = regexp.MustCompile(`// expect: ?(.*)$`)
	exitRe   = regexp.MustCompile(`^// exit: (\d+)`)
	stderrRe = regexp.MustCompile(`^// stderr: (.+)$`)
	opsRe    = regexp.MustCompile(`^// ops: (.+)$`)
)

type expectations struct {
	stdout []string
	exit   int
	stderr []string
	// ops solo lo usan los scripts de tests/bytecode.
	ops []string
}

func parseExpectations(t *testing.T, path string) expectations {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var e expectations
	for _, line := range strings.Split(string(src), "\n") {
		if m := expectRe.FindStringSubmatch(line); m != nil {
			e.stdout = append(e.stdout, m[1])
		}
		trimmed := strings.TrimSpace(line)
		if m := exitRe.FindStringSubmatch(trimmed); m != nil {
			e.exit, _ = strconv.Atoi(m[1])
		}
		if m := stderrRe.FindStringSubmatch(trimmed); m != nil {
			e.stderr = append(e.stderr, m[1])
		}
		if m := opsRe.FindStringSubmatch(trimmed); m != nil {
			e.ops = strings.Fields(m[1])
		}
	}
	return e
}

func splitLines(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func TestScripts(t *testing.T) {
	files, err := filepath.Glob("lox/*/*.lox")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no se encontraron scripts en tests/lox")
	}
	for _, path := range files {
		name := strings.TrimSuffix(strings.TrimPrefix(path, "lox/"), ".lox")
		t.Run(name, func(t *testing.T) {
			want := parseExpectations(t, path)
			got := runGlox(t, path)

			if got.exit != want.exit {
				t.Errorf("código de salida = %d, se esperaba %d\nstderr:\n%s", got.exit, want.exit, got.stderr)
			}
			gotLines := splitLines(got.stdout)
			if strings.Join(gotLines, "\n") != strings.Join(want.stdout, "\n") {
				t.Errorf("stdout distinto\n--- obtenido\n%s\n--- esperado\n%s", strings.Join(gotLines, "\n"), strings.Join(want.stdout, "\n"))
			}
			for _, fragment := range want.stderr {
				if !strings.Contains(got.stderr, fragment) {
					t.Errorf("stderr no contiene %q\nstderr:\n%s", fragment, got.stderr)
				}
			}
			if len(want.stderr) == 0 && got.stderr != "" {
				t.Errorf("stderr inesperado:\n%s", got.stderr)
			}
		})
	}
}

// Los programas de ejemplo tienen que seguir corriendo igual: su salida
// completa queda congelada en tests/golden (go test ./tests -update la regenera).
func TestExamples(t *testing.T) {
	files, err := filepath.Glob("../examples/*.lox")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		name := strings.TrimSuffix(filepath.Base(path), ".lox")
		t.Run(name, func(t *testing.T) {
			got := runGlox(t, path)
			if got.exit != 0 || got.stderr != "" {
				t.Fatalf("salió con código %d\nstderr:\n%s", got.exit, got.stderr)
			}
			golden := filepath.Join("golden", name+".out")
			if *update {
				if err := os.WriteFile(golden, []byte(got.stdout), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("falta %s (correr go test ./tests -update): %v", golden, err)
			}
			if got.stdout != string(want) {
				t.Errorf("la salida cambió respecto de %s\n--- obtenido\n%s", golden, got.stdout)
			}
		})
	}
}

// Los scripts de benchmark declaran su resultado con `// resultado: N`; así
// una optimización futura no puede "ganar tiempo" dando un resultado distinto.
func TestBenchmarkScripts(t *testing.T) {
	files, err := filepath.Glob("../benchmarks/*.lox")
	if err != nil {
		t.Fatal(err)
	}
	resultRe := regexp.MustCompile(`(?m)^// resultado: (.+)$`)
	for _, path := range files {
		name := strings.TrimSuffix(filepath.Base(path), ".lox")
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			m := resultRe.FindSubmatch(src)
			if m == nil {
				t.Fatalf("%s no declara `// resultado:`", path)
			}
			got := runGlox(t, path)
			if got.exit != 0 {
				t.Fatalf("salió con código %d\nstderr:\n%s", got.exit, got.stderr)
			}
			lines := splitLines(got.stdout)
			if len(lines) == 0 {
				t.Fatal("no imprimió nada")
			}
			last := lines[len(lines)-1]
			if !sameValue(last, string(m[1])) {
				t.Errorf("imprimió %q, se esperaba %s", last, m[1])
			}
		})
	}
}

// sameValue compara numéricamente si los dos lados son números (glox imprime
// 4.99995e+09 donde otras implementaciones imprimen 4999950000).
func sameValue(got, want string) bool {
	g, errG := strconv.ParseFloat(got, 64)
	w, errW := strconv.ParseFloat(want, 64)
	if errG == nil && errW == nil {
		return g == w
	}
	return got == want
}

// El REPL mantiene el estado entre líneas y no corta ante un error.
func TestREPL(t *testing.T) {
	cmd := exec.Command(gloxBin)
	cmd.Stdin = strings.NewReader("var a = 1;\nfun inc() { a = a + 1; return a; }\nprint nope;\nprint inc();\nprint a;\n")
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		t.Fatalf("el REPL terminó con error: %v", err)
	}
	stdout := strings.ReplaceAll(out.String(), "> ", "")
	if got := splitLines(stdout); strings.Join(got, ",") != "2,2" {
		t.Errorf("stdout = %q, se esperaba 2 y 2", stdout)
	}
	if !strings.Contains(errOut.String(), "variable no definida 'nope'") {
		t.Errorf("stderr = %q", errOut.String())
	}
}

func TestModoScanning(t *testing.T) {
	got := runGlox(t, "--scanning", "lox/expresiones/comparaciones.lox")
	if got.exit != 0 {
		t.Fatalf("código %d: %s", got.exit, got.stderr)
	}
	for _, want := range []string{"PRINT", "LESS_EQUAL", "NUMBER", "EOF"} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("la salida de --scanning no contiene %s:\n%s", want, got.stdout)
		}
	}
}

func TestModoParsing(t *testing.T) {
	got := runGlox(t, "--parsing", "lox/expresiones/aritmetica.lox")
	if got.exit != 0 {
		t.Fatalf("código %d: %s", got.exit, got.stderr)
	}
	if !strings.Contains(got.stdout, "(+ 1 (* 2 3))") {
		t.Errorf("la salida de --parsing no tiene el árbol esperado:\n%s", got.stdout)
	}
}

// --disassemble compila un programa, muestra el bytecode y lo ejecuta en la
// VM.
func TestModoDisassemble(t *testing.T) {
	path := filepath.Join(t.TempDir(), "expr.lox")
	if err := os.WriteFile(path, []byte("print 2 + 3 * 4;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := runGlox(t, "--disassemble", path)
	if got.exit != 0 || got.stderr != "" {
		t.Fatalf("código %d\nstderr:\n%s", got.exit, got.stderr)
	}
	want := strings.Join([]string{
		"== COMPILETIME ==",
		"0000    1 OP_CONSTANT         0 '2'",
		"0002    | OP_CONSTANT         1 '3'",
		"0004    | OP_CONSTANT         2 '4'",
		"0006    | OP_MULTIPLY",
		"0007    | OP_ADD",
		"0008    | OP_PRINT",
		"0009    | OP_RETURN",
		"== RUNTIME ==",
		"14",
		"",
	}, "\n")
	if got.stdout != want {
		t.Errorf("salida de --disassemble distinta\n--- obtenido\n%s--- esperado\n%s", got.stdout, want)
	}
}

// splitDisassemble separa la salida de --disassemble en sus dos secciones:
// los nombres de las instrucciones del listado y las líneas del resultado.
func splitDisassemble(t *testing.T, stdout string) (ops, result []string) {
	t.Helper()
	lines := splitLines(stdout)
	if len(lines) == 0 || lines[0] != "== COMPILETIME ==" {
		t.Fatalf("la salida no empieza con el listado del bytecode:\n%s", stdout)
	}
	for i, line := range lines[1:] {
		if line == "== RUNTIME ==" {
			return ops, lines[i+2:]
		}
		// Cada línea del listado es: offset, línea del fuente (o '|'),
		// instrucción y, si tiene, su operando.
		fields := strings.Fields(line)
		if len(fields) < 3 {
			t.Fatalf("línea del listado con formato inesperado: %q", line)
		}
		ops = append(ops, fields[2])
	}
	t.Fatalf("la salida no tiene la sección == RUNTIME ==:\n%s", stdout)
	return nil, nil
}

func TestBytecodeScripts(t *testing.T) {
	files, err := filepath.Glob("bytecode/*.lox")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no se encontraron scripts en tests/bytecode")
	}
	for _, path := range files {
		name := strings.TrimSuffix(filepath.Base(path), ".lox")
		t.Run(name, func(t *testing.T) {
			want := parseExpectations(t, path)
			got := runGlox(t, "--disassemble", path)

			if got.exit != want.exit {
				t.Errorf("código de salida = %d, se esperaba %d\nstderr:\n%s", got.exit, want.exit, got.stderr)
			}
			for _, fragment := range want.stderr {
				if !strings.Contains(got.stderr, fragment) {
					t.Errorf("stderr no contiene %q\nstderr:\n%s", fragment, got.stderr)
				}
			}
			if len(want.stderr) == 0 && got.stderr != "" {
				t.Errorf("stderr inesperado:\n%s", got.stderr)
			}

			// Un error estático corta antes de generar bytecode: no se
			// muestra ni se ejecuta nada.
			if want.exit == 65 {
				if got.stdout != "" {
					t.Errorf("con un error estático no se esperaba nada en stdout:\n%s", got.stdout)
				}
				return
			}

			ops, result := splitDisassemble(t, got.stdout)
			if want.ops != nil && strings.Join(ops, " ") != strings.Join(want.ops, " ") {
				t.Errorf("instrucciones distintas\n--- obtenido\n%s\n--- esperado\n%s", strings.Join(ops, " "), strings.Join(want.ops, " "))
			}
			if strings.Join(result, "\n") != strings.Join(want.stdout, "\n") {
				t.Errorf("resultado distinto\n--- obtenido\n%s\n--- esperado\n%s", strings.Join(result, "\n"), strings.Join(want.stdout, "\n"))
			}
		})
	}
}

// En el REPL, --disassemble trata cada línea como una entrada aparte: un
// error en una no corta la sesión ni afecta a las que siguen, y una línea
// vacía no hace nada.
func TestDisassembleREPL(t *testing.T) {
	cmd := exec.Command(gloxBin, "--disassemble")
	cmd.Stdin = strings.NewReader("var a = 1;\nprint a + 2;\n\n1 / 0;\n1 +;\nwhile (true) print 1;\nprint 3 * 3;\n")
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		t.Fatalf("el REPL terminó con error: %v", err)
	}

	stdout := strings.ReplaceAll(out.String(), "> ", "")
	want := strings.Join([]string{
		"== COMPILETIME ==",
		"0000    1 OP_CONSTANT         0 '1'",
		"0002    | OP_DEFINE_GLOBAL    1 'a'",
		"0004    | OP_RETURN",
		"== RUNTIME ==",
		"== COMPILETIME ==",
		"0000    1 OP_GET_GLOBAL       0 'a'",
		"0002    | OP_CONSTANT         1 '2'",
		"0004    | OP_ADD",
		"0005    | OP_PRINT",
		"0006    | OP_RETURN",
		"== RUNTIME ==",
		"3",
		// 1 / 0;: se compila bien y falla recién al ejecutar.
		"== COMPILETIME ==",
		"0000    1 OP_CONSTANT         0 '1'",
		"0002    | OP_CONSTANT         1 '0'",
		"0004    | OP_DIVIDE",
		"0005    | OP_POP",
		"0006    | OP_RETURN",
		"== RUNTIME ==",
		// `1 +;` y `while (true) print 1;` son errores estáticos: no dejan nada en stdout.
		"== COMPILETIME ==",
		"0000    1 OP_CONSTANT         0 '3'",
		"0002    | OP_CONSTANT         1 '3'",
		"0004    | OP_MULTIPLY",
		"0005    | OP_PRINT",
		"0006    | OP_RETURN",
		"== RUNTIME ==",
		"9",
		"",
	}, "\n")
	if stdout != want {
		t.Errorf("stdout distinto\n--- obtenido\n%s--- esperado\n%s", stdout, want)
	}

	wantErrs := []string{
		"[línea 1] Error en tiempo de ejecución en '/': división por cero.",
		"[línea 1] Error de sintaxis en ';': se esperaba una expresión.",
		"[línea 1] Error de compilación: el compilador a bytecode todavía no soporta el bucle 'while'.",
	}
	if got := splitLines(errOut.String()); strings.Join(got, "\n") != strings.Join(wantErrs, "\n") {
		t.Errorf("stderr distinto\n--- obtenido\n%s\n--- esperado\n%s", errOut.String(), strings.Join(wantErrs, "\n"))
	}
}

// Un archivo sin ninguna expresión (vacío o solo con comentarios) no es un
// error: no hay nada que compilar.
func TestDisassembleEntradaVacia(t *testing.T) {
	for name, content := range map[string]string{
		"vacío":            "",
		"solo comentarios": "// nada por acá\n\n// ni por acá\n",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "vacio.lox")
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			got := runGlox(t, "--disassemble", path)
			if got.exit != 0 || got.stdout != "" || got.stderr != "" {
				t.Errorf("exit = %d, stdout = %q, stderr = %q; se esperaba 0 y nada en la salida", got.exit, got.stdout, got.stderr)
			}
		})
	}
}

// Los mismos programas dan lo mismo por los dos caminos del CLI: el
// tree-walk y la VM con --disassemble. Es la comparación
// de los tests unitarios de internal/bytecode, pero sobre el binario real.
func TestDisassembleCoincideConElTreeWalk(t *testing.T) {
	sources := []string{
		"print 2 + 3 * 4;",
		"print (2 + 3) * 4;",
		"print -(1.5 + 2.5) / 8;",
		"print 7 % 3 - 10 / 4;",
		"print 1 / 3;",
		`print "ab" + "cd";`,
		"print 1 / 0;",
		"print 10 % (2 - 2);",
		`print 1 + "a";`,
		`print -"a";`,
		`print "a" * (1 / 0);`,
		"var a = 1; { var a = 2; print a; } print a;",
		"{ var x = 10; x = x + 5; print x; }",
		`if (1 < 2) print "si"; else print "no";`,
		"print false and 1;",
		"print true or 1;",
		"print 1 < 2;",
		"print 1 == 1;",
		"print 1 != 2;",
	}

	for _, source := range sources {
		t.Run(source, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "prog.lox")
			if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
				t.Fatal(err)
			}

			treeWalk := runGlox(t, path)
			vm := runGlox(t, "--disassemble", path)

			if vm.exit != treeWalk.exit {
				t.Errorf("código de salida: VM %d, tree-walk %d", vm.exit, treeWalk.exit)
			}
			if vm.stderr != treeWalk.stderr {
				t.Errorf("stderr distinto\n--- VM\n%s--- tree-walk\n%s", vm.stderr, treeWalk.stderr)
			}
			_, result := splitDisassemble(t, vm.stdout)
			if strings.Join(result, "\n") != strings.Join(splitLines(treeWalk.stdout), "\n") {
				t.Errorf("resultado distinto\n--- VM\n%s\n--- tree-walk\n%s", strings.Join(result, "\n"), treeWalk.stdout)
			}
		})
	}
}

func TestUsoIncorrecto(t *testing.T) {
	if got := runGlox(t, "a.lox", "b.lox"); got.exit != 64 {
		t.Errorf("con dos archivos el código fue %d, se esperaba 64", got.exit)
	}
	if got := runGlox(t, "no-existe.lox"); got.exit == 0 {
		t.Error("un archivo inexistente no debería terminar con 0")
	}
}
