package main

import (
	"fmt"
	"os"

	entry "github.com/oneseIf/calories"
)

func main() {
	argLength := len(os.Args[1:])
	if argLength == 0 {
		// create summary and plots
		fmt.Println("summary")
		os.Exit(0)
	}
	if os.Args[1] == "log" {
		entry.Log("TEST.csv")
	}
}
