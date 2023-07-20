package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
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

	/* ---------- Database ----------- */
	// Create a new SQLite database
	db, err := sqlx.Connect("sqlite", "../../database/mydata.db")
	if err != nil {
		log.Println(err)
		return
	}
	defer db.Close()

	// Read SQL file
	sqlBytes, err := ioutil.ReadFile("../../database/sql/setup.sql")
	if err != nil {
		log.Println(err)
		return
	}

	sqlStr := string(sqlBytes)

	// Execute setup SQL file
	_, err = db.Exec(sqlStr)
	if err != nil {
		log.Println(err)
		return
	}
	/* ------------------------------- */

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
		log.Println("Usage: ./calories [add|delete|update|summary|start|stop]")
		return
	}

	switch os.Args[1] {
	case "log":
		if len(os.Args) < 3 {
			log.Println("Usage: ./calories log [weight|food|meal]")
			return
		}

		// Execute subcommand
		switch os.Args[2] {
		case "meal":
			// TODO
		case "food":
			// TODO
		case "weight":
			// TODO
		default:
			log.Println("Usage: ./calories log [weight|food|meal]")
			return
		}
	case "add":
		if len(os.Args) < 3 {
			log.Println("Usage: ./calories add [log|food|meal]")
			return
		}

		// Execute subcommand
		switch os.Args[2] {
		case "log": // TODO: remove case once ./cmd log is made
			err := c.Log(u, c.EntriesFilePath)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Print(logs.Table())
			return
		case "meal":
			// TODO
		case "food":
			// TODO
		default:
			log.Println("Usage: ./calories add [log|food|meal]")
			return
		}
	case "delete":
		if len(os.Args) < 3 {
			log.Println("Usage: ./calories delete [log|food|meal]")
			return
		}

		// Execute subcommand
		switch os.Args[2] {
		case "log":
			// TODO
		case "meal":
			// TODO
		case "food":
			// TODO
		default:
			log.Println("Usage: ./calories delete [log|food|meal]")
			return
		}
	case "update":
		if len(os.Args) < 3 {
			log.Println("Usage: ./calories update [user|log]")
			return
		}

		// Execute subcommand
		switch os.Args[2] {
		case "log":
			// TODO
		case "user":
			c.UpdateUserInfo(u)
		default:
			log.Println("Usage: ./calories update [user|log]")
			return
		}
	case "summary":
		if len(os.Args) < 3 {
			log.Println("Usage: ./calories summary [phase|user|diet]")
			return
		}

		// Execute subcommand
		switch os.Args[2] {
		case "phase":
			// Only call Summary with the active logs if a diet phase is active.
			if active {
				c.Summary(u, active_logs)
				return
			}
			log.Println("Diet is not active. Skipping summary.")
		case "diet":
			// TODO: give summary on foods ate
		case "user":
			c.PrintUserInfo(u)
		default:
			log.Println("Usage: ./calories summary [phase|user|diet]")
			return
		}
	case "start":
		if len(os.Args) < 3 {
			log.Println("Usage: ./calories start [phase]")
			return
		}

		// Execute subcommand
		switch os.Args[2] {
		case "phase":
			// TODO
		default:
			log.Println("Usage: ./calories start [phase]")
			return
		}

	case "stop":
		if len(os.Args) < 3 {
			log.Println("Usage: ./calories stop [phase]")
			return
		}

		// Execute subcommand
		switch os.Args[2] {
		case "phase":
			// TODO
		default:
			log.Println("Usage: ./calories stop [phase]")
			return
		}
	case "test": // TODO: REMOVE AFTER TESTING.
		c.Run(db)
	default:
		log.Println("Usage: ./calories [add|delete|update|summary|start|stop]")
	}

	return
}
