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

// checkInput checks if the user input is positive
func checkInput(n float64) error {
	if n < 0 {
		return errors.New("invalid number")
	}
	return nil
}

// promptCals prompts the user to enter caloric intake for the previous
// day.
func promptCals() (calories float64, err error) {
	fmt.Print("Enter caloric intake for the day: ")
	fmt.Scanln(&calories)

	return calories, checkInput(calories)
}

// Log appends a new entry to the csv file passed in as an agurment.
func Log(u *UserInfo, s string) error {
	// Get user weight.
	u.Weight = getWeight()

	// Get user calories for the day.
	cals, err := promptCals()
	if err != nil {
		return err
	}

	// Save updated user info.
	err = saveUserInfo(u)
	if err != nil {
		return err
	}

	// Get current date.
	d := time.Now()

	// Open file for append.
	f, err := os.OpenFile(s, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	if err != nil {
		log.Println(err)
		return err
	}

	// Append user calorie input to csv file.
	line := fmt.Sprintf("%.2f,%.2f,%s\n", u.Weight, cals, d.Format("2006-01-02"))
	_, err = f.WriteString(line)
	if err != nil {
		log.Println(err)
		return err
	}

	fmt.Println("Added entry.")

	return nil
}
