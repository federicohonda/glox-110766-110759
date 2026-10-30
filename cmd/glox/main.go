package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/federicohonda/glox-110766-110759/internal/bytecode"
	"github.com/federicohonda/glox-110766-110759/internal/interpreter"
	"github.com/federicohonda/glox-110766-110759/internal/parser"
	"github.com/federicohonda/glox-110766-110759/internal/resolver"
	"github.com/federicohonda/glox-110766-110759/internal/scanner"
	"github.com/federicohonda/glox-110766-110759/internal/token"
)

func main() {
	scanning := flag.Bool("scanning", false, "Escanear e imprimir la lista de tokens")
	parsing := flag.Bool("parsing", false, "Parsear e imprimir el AST de la expresión")
	disassemble := flag.Bool("disassemble", false, "Imprimir el bytecode antes de ejecutarlo en la VM")
	treeWalk := flag.Bool("tree-walk", false, "Ejecutar con el intérprete tree-walk (Bloque 1) en vez de la VM")
	flag.Parse()

	args := flag.Args()
	m := mode{*scanning, *parsing, *disassemble, *treeWalk}

	switch len(args) {
	case 0:
		runPrompt(m)
	case 1:
		runFile(args[0], m)
	default:
		fmt.Fprintln(os.Stderr, "Uso: glox [--scanning] [--parsing] [--disassemble] [--tree-walk] [script]")
		os.Exit(64)
	}
}

// mode son los flags del CLI. Sin ninguno, el programa corre en la VM.
type mode struct {
	scanning, parsing, disassemble, treeWalk bool
}

func runFile(path string, m mode) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	interp := interpreter.New()
	vm := bytecode.NewVM(os.Stdout)
	hadError, hadRuntimeError := run(string(bytes), m, interp, vm)
	if hadError {
		os.Exit(65)
	}
	if hadRuntimeError {
		os.Exit(70)
	}
}

func runPrompt(m mode) {
	sc := bufio.NewScanner(os.Stdin)
	interp := interpreter.New()
	vm := bytecode.NewVM(os.Stdout)
	for {
		fmt.Print("> ")
		if !sc.Scan() {
			break
		}
		line := sc.Text()
		run(line, m, interp, vm)
	}
}

func run(source string, m mode, interp *interpreter.Interpreter, vm *bytecode.VM) (hadError bool, hadRuntimeError bool) {
	sc := scanner.New(source)
	tokens, errs := sc.Scan()

	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, err)
		}
	}

	if m.scanning {
		for _, tok := range tokens {
			fmt.Println(tok)
		}
	}

	if len(errs) > 0 {
		return true, false
	}

	if m.scanning {
		return false, false
	}

	if !m.parsing && !m.treeWalk {
		return runBytecode(tokens, vm, m.disassemble)
	}

	stmts, parseErrs := parser.New(tokens).Parse()
	if len(parseErrs) > 0 {
		for _, err := range parseErrs {
			fmt.Fprintln(os.Stderr, err)
		}
		return true, false
	}

	if m.parsing {
		for _, stmt := range stmts {
			fmt.Println(stmt)
		}
		return false, false
	}

	locals, resolveErrs := resolver.Resolve(stmts)
	if len(resolveErrs) > 0 {
		for _, err := range resolveErrs {
			fmt.Fprintln(os.Stderr, err)
		}
		return true, false
	}
	interp.Resolve(locals)

	if err := interp.Interpret(stmts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return false, true
	}

	return false, false
}

// runBytecode es el pipeline por defecto: compila a bytecode y lo ejecuta en
// la VM. Con --disassemble, antes muestra el listado del chunk.
func runBytecode(tokens []token.Token, vm *bytecode.VM, disassemble bool) (hadError bool, hadRuntimeError bool) {
	// Solo el EOF: una entrada vacía no hace nada, igual que un programa vacío.
	if len(tokens) == 1 {
		return false, false
	}

	stmts, parseErrs := parser.New(tokens).Parse()
	if len(parseErrs) > 0 {
		for _, err := range parseErrs {
			fmt.Fprintln(os.Stderr, err)
		}
		return true, false
	}

	_, resolveErrs := resolver.Resolve(stmts)
	if len(resolveErrs) > 0 {
		for _, err := range resolveErrs {
			fmt.Fprintln(os.Stderr, err)
		}
		return true, false
	}

	chunk, err := bytecode.CompileProgram(stmts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true, false
	}
	if disassemble {
		fmt.Print(chunk.Disassemble("COMPILETIME"))
		fmt.Println("== RUNTIME ==")
	}
	_, err = vm.Run(chunk)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return false, true
	}

	return false, false
}
