package entry

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"
)

func checkInput(num float32) error {
	if 0 > num || num > 30000 {
		return errors.New("invalid number")
	}
	return nil
}

func promptMass() (mass float32, err error) {
	// get body mass
	fmt.Print("Enter mass in lbs: ")
	fmt.Scanln(&mass)

	return mass, checkInput(mass)
}

func promptCals() (calories float32, err error) {
	// get caloric intake
	fmt.Print("Enter caloric intake for the day: ")
	fmt.Scanln(&calories)

	return calories, checkInput(calories)
}

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

func Log(s string) {
	mass, calories := Prompt()

	// get current date
	d := time.Now()

	// append user calorie input to csv file
	f, err := os.OpenFile(s, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()

	if err != nil {
		log.Println(err)
	}

	line := fmt.Sprintf("%s, %f, %f\n", d.Format("2006-01-02"), mass, calories)
	_, err = f.WriteString(line)
	if err != nil {
		log.Println(err)
	}
}
