package ui

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ericstrs/bite"
	"github.com/jmoiron/sqlx"
)

const (
	logUsage = `USAGE

	bite log food   - Log food.
	bite log meal   - Log meal.
	bite log weight - Log weight.
	bite log update [weight|food]     - Update food or weight log.
	bite log delete [weight|food]     - Delete food or weight log.
	bite log show   [all|weight|food] - Shows food and weight log and full log.
`
	createUsage = `USAGE

	bite create food - Create new food.
	bite create meal - Create new meal.
`
	deleteUsage = `USAGE

	bite delete food - Delete existing food.
	bite delete meal - Delete existing meal.
`
	updateUsage = `USAGE

	bite update food - Update food information.
	bite update weight - Update user information.
`
	summaryUsage = `USAGE

	bite summary phase - Print phase summary.
	bite summary diet  - Print diet summary.
	bite summary user  - Print user summary.
`
	stopUsage = `USAGE

	bite stop phase - Stop current phase.
`
)

func LogCmd(args []string) error {
	n := len(args)
	if n < 3 {
		printUsageExit(`ERROR: Not enough arguments`, logUsage)
	}
	dbPath := os.Getenv("BITE_DB_PATH")
	if dbPath == "" {
		log.Fatal("Environment variable BITE_DB_PATH must be set")
	}
	db, err := sqlx.Connect("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	c, err := bite.Config(db)
	if err != nil {
		return fmt.Errorf("ERROR: reading config: %v", err)
	}

	switch strings.ToLower(args[2]) {
	case `meal`:
		if err := NewSearchUI(db, "", `meal`).Run(); err != nil {
			return fmt.Errorf("couldn't run search ui: %v", err)
		}
		if err := SummaryCmd([]string{`zet`, `summary`, `diet`, `day`}); err != nil {
			return fmt.Errorf("couldn't get daily summary: %v", err)
		}
	case `food`:
		if err := NewSearchUI(db, "", `food`).Run(); err != nil {
			return fmt.Errorf("couldn't run search ui: %v", err)
		}
		if err := SummaryCmd([]string{`zet`, `summary`, `diet`, `day`}); err != nil {
			return fmt.Errorf("couldn't get daily summary: %v", err)
		}
	case `weight`:
		if err := bite.LogWeight(c, db); err != nil {
			return err
		}
	case `update`:
		if n < 4 {
			printUsageExit(`ERROR: Not enough arguments`, logUsage)
		}
		switch strings.ToLower(args[3]) {
		case `food`:
			if err := bite.UpdateFoodLog(db); err != nil {
				return err
			}
		case `weight`:
			if err := bite.UpdateWeightLog(db, c); err != nil {
				return err
			}
		default:
			printUsageExit(`ERROR: Incorrect argument`, logUsage)
		}
	case `delete`:
		if n < 4 {
			printUsageExit(`ERROR: Not enough arguments`, logUsage)
		}
		switch strings.ToLower(args[3]) {
		case `food`:
			if err := bite.DeleteFoodEntry(db); err != nil {
				return err
			}
		case `weight`:
			if err := bite.DeleteWeightEntry(db); err != nil {
				return err
			}
		default:
			printUsageExit(`ERROR: Incorrect argument`, logUsage)
		}
	case `show`:
		if n < 4 {
			printUsageExit(`ERROR: Not enough arguments`, logUsage)
		}
		switch strings.ToLower(args[3]) {
		case `all`:
			entries, err := bite.AllEntries(db)
			if err != nil {
				return err
			}
			bite.PrintEntries(*entries)
		case `food`:
			if err := bite.ShowFoodLog(db); err != nil {
				return err
			}
		case `weight`:
			if err := bite.ShowWeightLog(db); err != nil {
				return err
			}
		default:
			printUsageExit(`ERROR: Incorrect argument`, logUsage)
		}
	case `help`:
		fmt.Printf(logUsage)
	default:
		printUsageExit(`ERROR: Incorrect argument`, logUsage)
	}
	return nil
}

func CreateCmd(args []string) error {
	n := len(args)
	if n < 3 {
		printUsageExit(`ERROR: Not enough arguments`, createUsage)
	}
	dbPath := os.Getenv(`BITE_DB_PATH`)
	if dbPath == "" {
		log.Fatal("Environment variable BITE_DB_PATH must be set")
	}
	db, err := sqlx.Connect(`sqlite`, dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	switch strings.ToLower(args[2]) {
	case `meal`:
		if err := bite.CreateAddMeal(db); err != nil {
			return err
		}
	case `food`:
		if err := bite.CreateAddFood(db); err != nil {
			return err
		}
	case `help`:
		fmt.Printf(createUsage)
	default:
		printUsageExit(`ERROR: Incorrect argument`, createUsage)
	}
	return nil
}

func DeleteCmd(args []string) error {
	n := len(args)
	if n < 3 {
		printUsageExit(`ERROR: Not enough arguments`, deleteUsage)
	}
	dbPath := os.Getenv(`BITE_DB_PATH`)
	if dbPath == "" {
		log.Fatal("Environment variable BITE_DB_PATH must be set")
	}
	db, err := sqlx.Connect(`sqlite`, dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	switch strings.ToLower(args[2]) {
	case `meal`:
		if err := bite.SelectDeleteMeal(db); err != nil {
			return err
		}
	case `food`:
		if err := bite.SelectDeleteFood(db); err != nil {
			return err
		}
	case `help`:
		fmt.Printf(deleteUsage)
	default:
		printUsageExit(`ERROR: Incorrect argument`, deleteUsage)
	}
	return nil
}

func UpdateCmd(args []string) error {
	n := len(args)
	if n < 3 {
		printUsageExit(`ERROR: Not enough arguments`, updateUsage)
	}
	dbPath := os.Getenv(`BITE_DB_PATH`)
	if dbPath == "" {
		log.Fatal("Environment variable BITE_DB_PATH must be set")
	}
	db, err := sqlx.Connect(`sqlite`, dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	c, err := bite.Config(db)
	if err != nil {
		return fmt.Errorf("ERROR: reading config: %v", err)
	}

	switch strings.ToLower(args[2]) {
	case `user`:
		if err := bite.UpdateUserInfo(db, c); err != nil {
			return err
		}
	case `food`:
		if err := bite.UpdateFood(db); err != nil {
			return err
		}
	case `meal`:
		if len(os.Args) < 4 {
			printUsageExit(`ERROR: Not enough arguments`, updateUsage)
		}
		switch strings.ToLower(args[3]) {
		case `add`: // Adds a food to an existing meal.
			if err := bite.PromptAddMealFood(db); err != nil {
				return err
			}
		case `delete`: // Deletes a food from an existing meal.
			if err := bite.SelectDeleteFoodMealFood(db); err != nil {
				return err
			}
		default:
			printUsageExit(`ERROR: Incorrect argument`, updateUsage)
		}
	case `help`:
		fmt.Printf(updateUsage)
	default:
		printUsageExit(`ERROR: Incorrect argument`, updateUsage)
	}
	return nil
}

func SummaryCmd(args []string) error {
	n := len(args)
	if n < 3 {
		printUsageExit(`ERROR: Not enough arguments`, summaryUsage)
	}
	dbPath := os.Getenv(`BITE_DB_PATH`)
	if dbPath == "" {
		log.Fatal("Environment variable BITE_DB_PATH must be set")
	}
	db, err := sqlx.Connect(`sqlite`, dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	c, err := bite.Config(db)
	if err != nil {
		return fmt.Errorf("ERROR: reading config: %v", err)
	}

	// Execute subcommand
	switch strings.ToLower(args[2]) {
	case `phase`:
		status, err := bite.CheckPhaseStatus(db, c)
		if err != nil {
			return err
		}

		// Read user entries.
		entries, err := bite.AllEntries(db)
		if err != nil {
			return err
		}

		var activeLog *[]bite.Entry
		// If there is an active diet,
		if status == "active" {
			// Subset the log for the active diet phase.
			activeLog = bite.ValidLog(c, entries)

			// Get user progress.
			if err := bite.CheckProgress(db, c, activeLog); err != nil {
				return err
			}
		}

		// Only call Summary with the active logs if a diet phase is active.
		if status != `active` {
			return errors.New("diet is not active. Skipping summary.")
		}
		bite.Summary(c, activeLog)
	case `diet`:
		if n < 4 {
			printUsageExit(`ERROR: Not enough arguments`, summaryUsage)
		}
		switch strings.ToLower(args[3]) {
		case `all`:
			if err := bite.FoodLogSummary(db); err != nil {
				return err
			}
		case `day`:
			if err := bite.FoodLogSummaryDay(db, c); err != nil {
				return err
			}
		default:
			printUsageExit(`ERROR: Incorrect argument`, summaryUsage)
		}
	case `user`:
		bite.PrintUserInfo(c)
	case `help`:
		fmt.Printf(summaryUsage)
	default:
		printUsageExit(`ERROR: Incorrect argument`, summaryUsage)
	}
	return nil
}

func StopCmd(args []string) error {
	n := len(args)
	if n < 3 {
		printUsageExit(`ERROR: Not enough arguments`, stopUsage)
	}
	dbPath := os.Getenv(`BITE_DB_PATH`)
	if dbPath == "" {
		log.Fatal("Environment variable BITE_DB_PATH must be set")
	}
	db, err := sqlx.Connect(`sqlite`, dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	c, err := bite.Config(db)
	if err != nil {
		return fmt.Errorf("ERROR: Couldn't read config: %v", err)
	}

	switch strings.ToLower(os.Args[2]) {
	case "phase":
		if err := bite.StopPhase(db, c); err != nil {
			return err
		}
	case `help`:
		fmt.Printf(stopUsage)
	default:
		printUsageExit(`ERROR: Incorrect argument`, stopUsage)
	}
	return nil
}

// printUsageExit prints error message and usage statement, then exits
// the program with error code 1.
func printUsageExit(m, s string) {
	fmt.Fprintln(os.Stderr, m)
	fmt.Fprintf(os.Stderr, s)
	os.Exit(1)
}
