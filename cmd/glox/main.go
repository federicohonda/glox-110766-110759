package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/federicohonda/glox-110766-110759/internal/interpreter"
	"github.com/federicohonda/glox-110766-110759/internal/parser"
	"github.com/federicohonda/glox-110766-110759/internal/resolver"
	"github.com/federicohonda/glox-110766-110759/internal/scanner"
)

func main() {
	scanning := flag.Bool("scanning", false, "Escanear e imprimir la lista de tokens")
	parsing := flag.Bool("parsing", false, "Parsear e imprimir el AST de la expresión")
	flag.Parse()

	args := flag.Args()

	switch len(args) {
	case 0:
		runPrompt(*scanning, *parsing)
	case 1:
		runFile(args[0], *scanning, *parsing)
	default:
		fmt.Fprintln(os.Stderr, "Uso: glox [--scanning] [--parsing] [script]")
		os.Exit(64)
	}
}

func runFile(path string, scanning, parsing bool) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	interp := interpreter.New()
	hadError, hadRuntimeError := run(string(bytes), scanning, parsing, interp)
	if hadError {
		os.Exit(65)
	}
	if hadRuntimeError {
		os.Exit(70)
	}
}

func runPrompt(scanning, parsing bool) {
	sc := bufio.NewScanner(os.Stdin)
	interp := interpreter.New()
	for {
		fmt.Print("> ")
		if !sc.Scan() {
			break
		}
		line := sc.Text()
		run(line, scanning, parsing, interp)
	}
}

func run(source string, scanning, parsing bool, interp *interpreter.Interpreter) (hadError bool, hadRuntimeError bool) {
	sc := scanner.New(source)
	tokens, errs := sc.Scan()

	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, err)
		}
	}

	if scanning {
		for _, tok := range tokens {
			fmt.Println(tok)
		}
	}

	if len(errs) > 0 {
		return true, false
	}

	if scanning {
		return false, false
	}

	stmts, parseErrs := parser.New(tokens).Parse()
	if len(parseErrs) > 0 {
		for _, err := range parseErrs {
			fmt.Fprintln(os.Stderr, err)
		}
		return true, false
	}

	if parsing {
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
