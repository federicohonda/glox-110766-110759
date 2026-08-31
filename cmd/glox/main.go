package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]

	switch len(args) {
	case 0:
		runPrompt()
	case 1:
		runFile(args[0])
	default:
		fmt.Fprintln(os.Stderr, "Uso: glox [script]")
		os.Exit(64)
	}
}

func runFile(path string) {
	_, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runPrompt() {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		_ = scanner.Text()
	}
}
