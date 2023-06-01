package calories

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rocketlaunchr/dataframe-go"
	"github.com/rocketlaunchr/dataframe-go/imports"
)

// ReadEntries reads user entries from CSV file into a dataframe.
func ReadEntries() (*dataframe.DataFrame, error) {
	// Does entries file exist?
	if _, err := os.Stat(EntriesFilePath); os.IsNotExist(err) {
		log.Println("ERROR: Entries file not found.")
		return nil, err
	}

	// Open entries file
	csvfile, err := os.Open(EntriesFilePath)
	if err != nil {
		log.Printf("ERROR: Couldn't open %s\n", EntriesFilePath)
		return nil, err
	}
	defer csvfile.Close()

	// Read entries from CSV into a dataframe.
	ctx := context.TODO()
	logs, err := imports.LoadFromCSV(ctx, csvfile)
	if err != nil {
		log.Printf("ERROR: Couldn't read %s\n", EntriesFilePath)
		return nil, err
	}

	return logs, nil
}

// checkInput checks if the user input is between 0 and 30,000.
func checkInput(n float32) error {
	if 0 > n || n > 30000 {
		return errors.New("invalid number")
	}
	return nil
}

// promptMass prompts the user to enter their mass.
func promptMass() (mass float32, err error) {
	fmt.Print("Enter mass in lbs: ")
	fmt.Scanln(&mass)

	return mass, checkInput(mass)
}

// promptCals prompts the user to enter caloric intake for the previous
// day.
func promptCals() (calories float32, err error) {
	fmt.Print("Enter caloric intake for the day: ")
	fmt.Scanln(&calories)

	return calories, checkInput(calories)
}

// promptMassCals prompts and returns user mass and caloric intake.
func promptMassCals() (mass float32, cals float32) {
	// TODO: replace if statements with switch statement
	//fmt.Println("Press 'q' to quit")

	for {
		mass, err := promptMass()
		if err != nil {
			fmt.Printf("Couldn't read mass: %s\n\n", err)
			continue
		}

		cals, err := promptCals()
		if err != nil {
			fmt.Printf("Couldn't read calories: %s\n\n", err)
			continue
		}

		return mass, cals
	}
}

// Log appends a new entry to the csv file passed in as an agurment.
func Log(s string) {
	// Prompt the user for mass and calorie info.
	mass, cals := promptMassCals()

	// Get current date.
	d := time.Now()

	// Open file for append.
	f, err := os.OpenFile(s, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	if err != nil {
		log.Println(err)
		return
	}

	// Append user calorie input to csv file.
	line := fmt.Sprintf("%.2f,%.2f,%s\n", mass, cals, d.Format("2006-01-02"))
	_, err = f.WriteString(line)
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println("Added entry.")
}
