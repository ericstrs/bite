package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"

	c "github.com/oneseIf/calories"
	"github.com/rocketlaunchr/dataframe-go/imports"
	"gopkg.in/yaml.v2"
)

const configFilePath = "./config.yaml"
const entriesFilePath = "./data.csv"

type UserInfo struct {
	Height        float64 `yaml:"height"`
	Age           int     `yaml:"age"`
	Gender        string  `yaml:"gender"`
	ActivityLevel string  `yaml:"activity_level"`
	Phase         string  `yaml:"phase"`
}

func saveUserInfo(userInfo UserInfo) error {
	data, err := yaml.Marshal(userInfo)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(configFilePath, data, 0644)
}

func promptUserInfo() UserInfo {
	var userInfo UserInfo

	fmt.Print("Enter height (cm): ")
	fmt.Scanln(&userInfo.Height)

	fmt.Print("Enter gender: ")
	fmt.Scanln(&userInfo.Gender)

	fmt.Print("Enter age: ")
	fmt.Scanln(&userInfo.Age)

	fmt.Print("Enter activity level (sedentary, light, moderate, active, very: ")
	fmt.Scanln(&userInfo.ActivityLevel)

	fmt.Print("Enter phase (cut, bulk, or maintain): ")
	fmt.Scanln(&userInfo.Phase)

	return userInfo
}

func main() {

	arg := os.Args[1]
	// Check if argument is log. Creating a new entry does not require
	// reading in user-info and user entries so we can return early.
	if arg == "log" {
		c.Log(entriesFilePath)
		return
	}

	var userInfo UserInfo

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		fmt.Println("Config file not found. Please provide required information:")
		// Prompt user for info.
		userInfo = promptUserInfo()

		// Save user info to config file.
		err := saveUserInfo(userInfo)
		if err != nil {
			log.Println("Failed to save user info:", err)
			return
		}
		fmt.Println("User info saved successfully.")
	} else {
		// Read YAML file.
		fp := configFilePath
		data, err := ioutil.ReadFile(fp)
		if err != nil {
			log.Printf("Error: Can't read file: %v\n", err)
			return
		}

		// Unmarshal YAML data into struct.
		userInfo = UserInfo{}
		err = yaml.Unmarshal(data, &userInfo)
		if err != nil {
			log.Printf("Error: Can't unmarshal YAML: %v\n", err)
			return
		}
		fmt.Println("User info loaded successful.")
	}

	if _, err := os.Stat(entriesFilePath); os.IsNotExist(err) {
		log.Println("Error: Entries file not found.")
		return
	}

	csvfile, err := os.Open(entriesFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer csvfile.Close()

	// Read entries from CSV into a dataframe.
	ctx := context.TODO()
	logs, err := imports.LoadFromCSV(ctx, csvfile)
	fmt.Print(logs.Table())

	switch arg {
	case "metrics":
		if logs.NRows() < 1 {
			log.Printf("Error: Not enough entries to produce metrics.\n")
			return
		}

		// Get most recent weight as a string.
		vals := logs.Series[0].Value(logs.NRows() - 1).(string)
		// Convert string to float64.
		weight, err := strconv.ParseFloat(vals, 64)
		if err != nil {
			fmt.Println("Failed to convert string to float64:", err)
			return
		}

		// Get BMR
		bmr := c.Mifflin(weight, userInfo.Height, userInfo.Age, userInfo.Gender)
		fmt.Printf("BMR: %.2f\n", bmr)

		// Get TDEE
		t := c.TDEE(bmr, userInfo.ActivityLevel)
		fmt.Printf("TDEE: %.2f\n", t)

		// Get suggested macro split
		protein, carbs, fats := c.Macros(weight, 0.4)
		fmt.Printf("Protein: %.2fg Carbs: %.2fg Fats: %.2fg\n", protein, carbs, fats)

		// Create plots
	case "summary":
		fmt.Println("Generating summary.")
		// Print day summary
		// Print week summary
		// Print month summary
	default:
		fmt.Println("Error: usage")
	}
	return
}
