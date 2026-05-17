package main

import (
	"fmt"
	"slices"
	"syscall/js"
)

func main() {
	fmt.Println("Go WebAssembly Initialized")

	// Register a function in the global JavaScript scope
	js.Global().Set("processData", js.FuncOf(processData))

	// Keep the program alive so functions remain callable
	select {}
}

func processData(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return "Error: No input provided"
	}

	input := args[0].String()
	fmt.Printf("Processing input: %s\n", input)

	// In a real demo, this is where you'd call your library.
	// We'll reverse the string as a simple placeholder logic.
	runes := []rune(input)
	slices.Reverse(runes)

	return string(runes)
}
