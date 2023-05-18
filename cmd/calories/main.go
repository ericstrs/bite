package main

import (
	"fmt"
	"io/ioutil"
	"os"

	entry "github.com/oneseIf/calories"
	"gopkg.in/yaml.v2"
)

type UserInfo struct {
	Height        int    `yaml:"height"`
	Age           int    `yaml:"age"`
	Gender        string `yaml:"gender"`
	ActivityLevel string `yaml:"activity_level"`
}

func main() {

	// Read YAML file
	fp := "user-info.yaml"
	data, err := ioutil.ReadFile(fp)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	// Unmarshal YAML data into struct
	userInfo := UserInfo{}
	err = yaml.Unmarshal(data, &userInfo)
	if err != nil {
		fmt.Printf("Error unmarshling YAML: %v\n", err)
		return
	}

	arg := os.Args[1]
	switch arg {
	case "metrics":
		// Get most recent mass. TODO: check if file exists.

		// Get BMR
		//bmr := Mifflin(

		// Get TDEE

		// Create plots
	case "log":
		entry.Log("TEST.csv")
	case "summary":
		fmt.Println("Generating summary.")
	default:
		fmt.Println("Error: usage")
	}
}
