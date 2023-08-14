package main

import (
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	b "github.com/oneseIf/bite"
)

func main() {
	var active_log *[]b.Entry

	// Connect to SQLite database
	db, err := sqlx.Connect("sqlite", "../../database/mydata.db")
	if err != nil {
		log.Println(err)
		return
	}
	defer db.Close()

	// Read user's config file.
	u, err := b.ReadConfig(db)
	if err != nil {
		return
	}

	// Read user entries.
	entries, err := b.GetAllEntries(db)
	if err != nil {
		return
	}

	status, err := b.CheckPhaseStatus(db, u)
	if err != nil {
		return
	}
	// If there is an active diet,
	if status == "active" {
		// Subset the log for the active diet phase.
		active_log = b.GetValidLog(u, entries)

		// Get user progress.
		err = b.CheckProgress(db, u, active_log)
		if err != nil {
			log.Println(err)
			return
		}
	}

	// Check if user has at least a single argument.
	if len(os.Args) < 2 {
		log.Println("Usage: ./bite [log|create|update|delete|summary|stop]")
		return
	}

	switch os.Args[1] {
	case "log":
		if len(os.Args) < 3 {
			log.Println("Usage: ./bite log [weight|food|meal|update|delete|show]")
			return
		}

		// Execute subcommand
		switch os.Args[2] {
		case "meal":
			b.LogMeal(db)
		case "food":
			b.LogFood(db)
		case "weight":
			b.LogWeight(u, db)
		case "update":
			if len(os.Args) < 4 {
				log.Println("Usage: ./bite log update [weight|food]")
				return
			}

			switch os.Args[3] {
			case "food":
				err := b.UpdateFoodLog(db)
				if err != nil {
					return
				}
			case "weight":
				err := b.UpdateWeightLog(db, u)
				if err != nil {
					return
				}
			default:
				log.Println("Usage: ./bite log update [weight|food]")
				return
			}
		case "delete":
			if len(os.Args) < 4 {
				log.Println("Usage: ./bite log delete [weight|food]")
				return
			}

			switch os.Args[3] {
			case "food":
				err := b.DeleteFoodEntry(db)
				if err != nil {
					return
				}
			case "weight":
				err := b.DeleteWeightEntry(db)
				if err != nil {
					return
				}
			default:
				log.Println("Usage: ./bite log delete [weight|food]")
				return
			}
		case "show":
			if len(os.Args) < 4 {
				log.Println("Usage: ./bite log show [all|weight|food]")
				return
			}

			switch os.Args[3] {
			case "all":
				b.PrintEntries(*entries)
			case "food":
				err := b.ShowFoodLog(db)
				if err != nil {
					log.Println(err)
					return
				}
			case "weight":
				err := b.ShowWeightLog(db)
				if err != nil {
					log.Println(err)
					return
				}
			default:
				log.Println("Usage: ./bite log show [all|weight|food]")
				return
			}
		default:
			log.Println("Usage: ./bite log [weight|food|meal|update|delete|show]")
			return
		}
	case "create":
		if len(os.Args) < 3 {
			log.Println("Usage: ./bite create [food|meal]")
			return
		}

		// Execute subcommand
		switch os.Args[2] {
		case "meal":
			err := b.CreateAndAddMeal(db)
			if err != nil {
				return
			}
		case "food":
			err := b.CreateAndAddFood(db)
			if err != nil {
				return
			}
		default:
			log.Println("Usage: ./bite create [food|meal]")
			return
		}
	case "delete":
		if len(os.Args) < 3 {
			log.Println("Usage: ./bite delete [food|meal]")
			return
		}

		// Execute subcommand
		switch os.Args[2] {
		case "meal":
			err := b.SelectAndDeleteMeal(db)
			if err != nil {
				return
			}
		case "food":
			err := b.SelectAndDeleteFood(db)
			if err != nil {
				return
			}
		default:
			log.Println("Usage: ./bite delete [food|meal]")
			return
		}
	case "update":
		if len(os.Args) < 3 {
			log.Println("Usage: ./bite update [user|food|meal]")
			return
		}

		// Execute subcommand
		switch os.Args[2] {
		case "user":
			if err := b.UpdateUserInfo(db, u); err != nil {
				return
			}
		case "food":
			if err := b.UpdateFood(db); err != nil {
				return
			}
		case "meal":
			if len(os.Args) < 4 {
				log.Println("Usage: ./bite update meal [add|delete]")
				return
			}

			switch os.Args[3] {
			case "add": // Adds a food to an existing meal.
				err := b.GetUserInputAddMealFood(db)
				if err != nil {
					return
				}
			case "delete": // Deletes a food from an existing meal.
				err := b.SelectAndDeleteFoodMealFood(db)
				if err != nil {
					return
				}
			default:
				log.Println("Usage: ./bite update meal [add|delete]")
				return
			}

		default:
			log.Println("Usage: ./bite update [user|food|meal]")
			return
		}
	case "summary":
		if len(os.Args) < 3 {
			log.Println("Usage: ./bite summary [phase|user|diet]")
			return
		}

		// Execute subcommand
		switch os.Args[2] {
		case "phase":
			// Only call Summary with the active logs if a diet phase is active.
			if status == "active" {
				b.Summary(u, active_log)
				return
			}
			log.Println("Diet is not active. Skipping summary.")
		case "diet":
			if len(os.Args) < 4 {
				log.Println("Usage: ./bite summary diet [all|day]")
				return
			}

			switch os.Args[3] {
			case "all":
				if err := b.FoodLogSummary(db); err != nil {
					return
				}
			case "day":
				if err := b.FoodLogSummaryDay(db, u); err != nil {
					return
				}
			default:
				log.Println("Usage: ./bite summary diet [all|day]")
				return
			}
		case "user":
			b.PrintUserInfo(u)
		default:
			log.Println("Usage: ./bite summary [phase|user|diet]")
			return
		}
	case "stop":
		if len(os.Args) < 3 {
			log.Println("Usage: ./bite stop [phase]")
			return
		}

		// Execute subcommand
		switch os.Args[2] {
		case "phase":
			if err := b.StopPhase(db, u); err != nil {
				return
			}
		default:
			log.Println("Usage: ./bite stop [phase]")
			return
		}
	default:
		log.Println("Usage: ./bite [log|create|update|delete|summary|stop]")
	}

	return
}
