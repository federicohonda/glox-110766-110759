package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/federicohonda/glox-110766-110759/internal/parser"
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

	hadError := run(string(bytes), scanning, parsing)
	if hadError {
		os.Exit(65)
	}
}

func runPrompt(scanning, parsing bool) {
	sc := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !sc.Scan() {
			break
		}
		line := sc.Text()
		run(line, scanning, parsing)
	}
}

func run(source string, scanning, parsing bool) bool {
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
		return true
	}

	if parsing {
		expr, err := parser.New(tokens).Parse()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return true
		}
		fmt.Println(expr)
	}

	return false
}
