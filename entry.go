package calories

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"
)

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

// Prompt returns the mass and calorie user input.
func Prompt() (mass float32, calories float32) {
	// TODO: replace if statements with switch statement
	//fmt.Println("Press 'q' to quit")

	for {
		mass, err := promptMass()
		if err != nil {
			fmt.Printf("Couldn't read mass: %s\n\n", err)
			continue
		}

		calories, err := promptCals()
		if err != nil {
			fmt.Printf("Couldn't read calories: %s\n\n", err)
			continue
		}

		return mass, calories
	}
}

// Log appends a new entry to the csv file passed in as an agurment.
func Log(s string) {
	// Prompt the user for mass and calorie info.
	mass, calories := Prompt()

	// Get current date.
	d := time.Now()

	// Open file for append.
	f, err := os.OpenFile(s, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()

	if err != nil {
		log.Println(err)
	}

	// Append user calorie input to csv file.
	line := fmt.Sprintf("%s, %f, %f\n", d.Format("2006-01-02"), mass, calories)
	_, err = f.WriteString(line)
	if err != nil {
		log.Println(err)
	}
}
