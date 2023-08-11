package main

import (
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	c "github.com/oneseIf/calories"
)

func main() {
	var active_log *[]c.Entry

	/* ---------- Database ----------- */
	// Create a new SQLite database
	db, err := sqlx.Connect("sqlite", "../../database/mydata.db")
	if err != nil {
		log.Println(err)
		return
	}
	defer db.Close()

	/* ------------------------------- */

	// Read user's config file.
	u, err := c.ReadConfig(db)
	if err != nil {
		return
	}

	// Read user entries.
	entries, err := c.GetAllEntries(db)
	if err != nil {
		return
	}

	status, err := c.CheckPhaseStatus(db, u)
	if err != nil {
		return
	}
	// If there is an active diet,
	if status == "active" {
		// Subset the log for the active diet phase.
		active_log = c.GetValidLog(u, entries)

		// Get user progress.
		err = c.CheckProgress(db, u, active_log)
		if err != nil {
			return
		}
	}

	// Check if user has at least a single argument.
	if len(os.Args) < 2 {
		log.Println("Usage: ./calories [log|add|delete|update|summary|start|stop]")
		return
	}

	switch os.Args[1] {
	case "log":
		if len(os.Args) < 3 {
			log.Println("Usage: ./calories log [weight|food|meal|update|delete|show]")
			return
		}

		// Execute subcommand
		switch os.Args[2] {
		case "meal":
			c.LogMeal(db)
		case "food":
			c.LogFood(db)
		case "weight":
			c.LogWeight(u, db)
		case "update":
			if len(os.Args) < 4 {
				log.Println("Usage: ./calories log update [weight|food]")
				return
			}

			switch os.Args[3] {
			case "food":
				err := c.UpdateFoodLog(db)
				if err != nil {
					return
				}
			case "weight":
				err := c.UpdateWeightLog(db, u)
				if err != nil {
					return
				}
			default:
				log.Println("Usage: ./calories log update [weight|food]")
				return
			}
		case "delete":
			if len(os.Args) < 4 {
				log.Println("Usage: ./calories log delete [weight|food]")
				return
			}

			switch os.Args[3] {
			case "food":
				err := c.DeleteFoodEntry(db)
				if err != nil {
					return
				}
			case "weight":
				err := c.DeleteWeightEntry(db)
				if err != nil {
					return
				}
			default:
				log.Println("Usage: ./calories log delete [weight|food]")
				return
			}
		case "show":
			if len(os.Args) < 4 {
				log.Println("Usage: ./calories log show [all|weight|food]")
				return
			}

			switch os.Args[3] {
			case "all":
				c.PrintEntries(*entries)
			case "food":
				err := c.ShowFoodLog(db)
				if err != nil {
					log.Println(err)
					return
				}
			case "weight":
				err := c.ShowWeightLog(db)
				if err != nil {
					log.Println(err)
					return
				}
			default:
				log.Println("Usage: ./calories log show [all|weight|food]")
				return
			}
		default:
			log.Println("Usage: ./calories log [weight|food|meal|update|delete|show]")
			return
		}
	case "add":
		if len(os.Args) < 3 {
			log.Println("Usage: ./calories add [food|meal]")
			return
		}

		// Execute subcommand
		switch os.Args[2] {
		case "meal":
			err := c.CreateAndAddMeal(db)
			if err != nil {
				return
			}
		case "food":
			err := c.CreateAndAddFood(db)
			if err != nil {
				return
			}
		default:
			log.Println("Usage: ./calories add [food|meal]")
			return
		}
	case "delete":
		if len(os.Args) < 3 {
			log.Println("Usage: ./calories delete [food|meal]")
			return
		}

		// Execute subcommand
		switch os.Args[2] {
		case "meal":
			err := c.SelectAndDeleteMeal(db)
			if err != nil {
				return
			}
		case "food":
			err := c.SelectAndDeleteFood(db)
			if err != nil {
				return
			}
		default:
			log.Println("Usage: ./calories delete [food|meal]")
			return
		}
	case "update":
		if len(os.Args) < 3 {
			log.Println("Usage: ./calories update [user|meal]")
			return
		}

		// Execute subcommand
		switch os.Args[2] {
		case "user":
			err := c.UpdateUserInfo(db, u)
			if err != nil {
				return
			}
		case "meal":
			if len(os.Args) < 4 {
				log.Println("Usage: ./calories update meal [add|delete]")
				return
			}

			switch os.Args[3] {
			case "add": // Adds a food to an existing meal.
				err := c.GetUserInputAddMealFood(db)
				if err != nil {
					return
				}
			case "delete": // Deletes a food from an existing meal.
				err := c.SelectAndDeleteFoodMealFood(db)
				if err != nil {
					return
				}
			default:
				log.Println("Usage: ./calories update meal [add|delete]")
				return
			}

		default:
			log.Println("Usage: ./calories update [user|meal]")
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
			if status == "active" {
				c.Summary(u, active_log)
				return
			}
			log.Println("Diet is not active. Skipping summary.")
		case "diet":
			if len(os.Args) < 4 {
				log.Println("Usage: ./calories summary diet [all|day]")
				return
			}

			switch os.Args[3] {
			case "all":
				if err := c.FoodLogSummary(db); err != nil {
					return
				}
			case "day":
				if err := c.FoodLogSummaryDay(db, u); err != nil {
					return
				}
			default:
				log.Println("Usage: ./calories summary diet [all|day]")
				return
			}
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
			if err := c.StopPhase(db, u); err != nil {
				return
			}
		default:
			log.Println("Usage: ./calories stop [phase]")
			return
		}
	default:
		log.Println("Usage: ./calories [log|add|delete|update|summary|start|stop]")
	}

	return
}
