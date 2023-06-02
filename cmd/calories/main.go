package main

import (
	"fmt"
	"log"
	"os"

	c "github.com/oneseIf/calories"
)

func main() {
	// Read user's config file.
	u, err := c.ReadConfig()
	if err != nil {
		return
	}

	// Read user entries.
	logs, err := c.ReadEntries()
	if err != nil {
		return
	}

	// Print out user entries.
	fmt.Print(logs.Table())

	// Check diet progress.
	err = c.CheckProgress(u, logs)
	if err != nil {
		return
	}

	arg := os.Args[1]
	switch arg {
	case "log":
		err := c.Log(u, c.EntriesFilePath)
		if err != nil {
			return
		}
		return
	case "summary":
		c.Summary()
	default:
		log.Println("Error: usage")
	}
	return
}
