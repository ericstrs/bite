/*
Bite is a command-line utility for managing diet phases and food logging.

USAGE

	bite [command]

COMMAND

	log     - Manages food, meal, and weight log.
	create  - Creates food or meal.
	delete  - Deletes food or meal.
	update  - Updates food, meal, or user information.
	summary - Provides phase, diet, and user summary.
	stop    - Stops a current phase.
*/
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ericstrs/bite/internal/ui"
)

const usage = `USAGE

	bite [command]

COMMANDS

	log     - Manages food, meal, and weight log.
	create  - Creates food or meal.
	delete  - Deletes food or meal.
	update  - Updates food, meal, or user information.
	summary - Provides phase, diet, and user summary.
	stop    - Stops a current phase.

DESCRIPTION

	Bite is a command-line utility for managing diet phases and food logging.

	Appending "help" after any command will print more command information.
`

func main() {
	if err := Run(); err != nil {
		log.Println(err)
	}
}

func Run() error {
	args := os.Args
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, `ERROR: Not enough arguments`)
		fmt.Fprintf(os.Stderr, usage)
		os.Exit(1)
	}

	/*
		dbPath := os.Getenv("BITE_DB_PATH")
		if dbPath == "" {
			log.Fatal("Environment variable BITE_DB_PATH must be set")
		}

		// Connect to SQLite database
		db, err := sqlx.Connect("sqlite", dbPath)
		if err != nil {
			return err
		}
		defer db.Close()

		// Read user's config file.
		u, err := b.Config(db)
		if err != nil {
			return err
		}

		status, err := b.CheckPhaseStatus(db, u)
		if err != nil {
			return err
		}

		// Read user entries.
		entries, err := b.GetAllEntries(db)
		if err != nil {
			return err
		}

		var activeLog *[]b.Entry
		// If there is an active diet,
		if status == "active" {
			// Subset the log for the active diet phase.
			activeLog = b.GetValidLog(u, entries)

			// Get user progress.
			err = b.CheckProgress(db, u, activeLog)
			if err != nil {
				return err
			}
		}
	*/

	switch strings.ToLower(args[1]) {
	case `log`:
		if err := ui.LogCmd(args); err != nil {
			return err
		}
	case `create`:
		if err := ui.CreateCmd(args); err != nil {
			return err
		}
	case `delete`:
		if err := ui.DeleteCmd(args); err != nil {
			return err
		}
	case `update`:
		if err := ui.UpdateCmd(args); err != nil {
			return err
		}
	case `summary`:
		if err := ui.SummaryCmd(args); err != nil {
			return err
		}
	case `stop`:
		if err := ui.StopCmd(args); err != nil {
			return err
		}
	case `help`:
		fmt.Printf(usage)
	default:
		fmt.Fprintln(os.Stderr, `ERROR: Incorrect argument.`)
		fmt.Fprintf(os.Stderr, usage)
		os.Exit(1)
	}
	return nil
}
