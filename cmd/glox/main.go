package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/federicohonda/glox-110766-110759/internal/scanner"
)

func main() {
	scanning := flag.Bool("scanning", false, "Escanear e imprimir la lista de tokens")
	flag.Parse()

	args := flag.Args()

	switch len(args) {
	case 0:
		runPrompt(*scanning)
	case 1:
		runFile(args[0], *scanning)
	default:
		fmt.Fprintln(os.Stderr, "Uso: glox [--scanning] [script]")
		os.Exit(64)
	}
}

func runFile(path string, scanning bool) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	hadError := run(string(bytes), scanning)
	if hadError {
		os.Exit(65)
	}
}

func runPrompt(scanning bool) {
	sc := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !sc.Scan() {
			break
		}
		line := sc.Text()
		run(line, scanning)
	}
}

func run(source string, scanning bool) bool {
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

	return len(errs) > 0
}
