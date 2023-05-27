package main

import (
	"log"
	"os"

	c "github.com/oneseIf/calories"
)

const ConfigFilePath = "./config.yaml"
const EntriesFilePath = "./data.csv"

func main() {
	c.ReadConfig()
	c.ReadEntries()
	//c.CheckProgress()

	arg := os.Args[1]
	switch arg {
	case "log":
		c.Log(EntriesFilePath)
		return
	case "summary":
		c.Summary()
	default:
		log.Println("Error: usage")
	}
	return
}
