package main

import (
	"fmt"
	"log"
	"os"

	c "github.com/oneseIf/calories"
	"github.com/rocketlaunchr/dataframe-go"
)

func main() {
	var active_logs *dataframe.DataFrame

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

	active, err := c.CheckPhaseStatus(u)
	if err != nil {
		return
	}
	// If there is an active diet,
	if active {
		// Obtain valid log indices for the active diet phase.
		indices := c.GetValidLogIndices(u, logs)
		// Subset the logs for the active diet phase.
		active_logs = c.Subset(logs, indices)

		err = c.CheckProgress(u, active_logs)
		if err != nil {
			return
		}
	}

	// Check if user has at least a single argument.
	if len(os.Args) < 2 {
		return
	}

	arg := os.Args[1]
	switch arg {
	case "log":
		err := c.Log(u, c.EntriesFilePath)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Print(logs.Table())
		return
	case "summary":
		// Only call Summary with the active logs if a diet phase is active.
		if active {
			c.Summary(u, active_logs)
			return
		}
		log.Println("Diet is not active. Skipping summary.")
	case "info":
		if len(os.Args) < 3 {
			c.PrintUserInfo(u)
			return
		}

		switch os.Args[2] {
		case "update":
			c.UpdateUserInfo(u)

		default:
			fmt.Printf("Unknown subcommand: %s\n", os.Args[2])
		}

	default:
		log.Println("Error: usage")
	}
	return
}
