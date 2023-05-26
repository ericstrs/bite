// Calories package provides dietary insight.
package calories

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

const configFilePath = "./config.yaml"
const entriesFilePath = "./data.csv"

type UserInfo struct {
	Gender        string    `yaml:"gender"`
	Weight        float64   `yaml:"weight"`
	Height        float64   `yaml:"height"`
	Age           int       `yaml:"age"`
	ActivityLevel string    `yaml:"activity_level"`
	Phase         PhaseInfo `yaml:"phase"`
}

type PhaseInfo struct {
	Name         string    `yaml:"name"`
	GoalWeight   float64   `yaml:"goal_weight"`
	WeeklyChange float64   `yaml:"weekly_change"`
	StartDate    time.Date `yaml:"start_date"`
	EndDate      time.Date `yaml:"end_date"`
	Duration     float64   `yaml:"duration"`
	Active       bool      `yaml:"active"`
}

// activity returns the scale based on the user's activity level.
func activity(a string) (float64, error) {
	activityMap := map[string]float64{
		"sedentary": 1.2,
		"light":     1.375,
		"moderate":  1.55,
		"active":    1.725,
		"very":      1.9,
	}

	value, f := activityMap[a]
	if !f {
		return -1, fmt.Errorf("unknown activity level: %s", a)
	}

	return value, nil
}

// Mifflin calculates and returnsthe Basal Metabolic Rate (BMR) which is based on
// weight (kg), height (cm), age (years), and gender.
func Mifflin(weight, height float64, age int, gender string) float64 {
	factor := 5
	if gender == "female" {
		factor = -151
	}

	// Convert lbs to kgs.
	weight = weight * 0.45359237

	var bmr float64
	bmr = (10 * weight) + (6.25 * height) - (5 * float64(age)) + float64(factor)
	return bmr
}

// TDEE calcuates the Total Daily Energy Expenditure (TDEE) based on the
// BMR and user's activity level.
func TDEE(bmr float64, a string) float64 {
	al, err := activity(a)
	if err != nil {
		fmt.Println("Error:", err)
		return -1
	}

	return bmr * al
}

// Macros calculates and returns the recommended macronutrients given
// user weight and desired fat percentage.
func Macros(weight, fatPercent float64) (float64, float64, float64) {
	protein := 1 * weight
	fats := fatPercent * weight
	remaining := (protein * 4) + (fats * 9)
	carbs := remaining / 4

	return protein, carbs, fats
}

// TODO:
// * Deal with errors
func Metrics() {
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
}

// Save userInfo to yaml config file
func saveUserInfo(u *UserInfo) error {
	data, err := yaml.Marshal(u)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(configFilePath, data, 0644)
}

// promptUserType prompts the user to enter desired diet phase.
func promptDietType(u *UserInfo) {
	// Prompt user for diet type
	fmt.Println("Step 2: Choose diet type.")

	fmt.Println("Fat loss (cut). Lose fat while losing weight and preserving muscle.")
	fmt.Println("Maintenance (maintain). Stay at your current weight.")
	fmt.Println("Muscle gain (bulk). Gain muscle while minimizing fat.")

	for {
		// Prompt user for diet phase.
		fmt.Print("Enter phase (cut, maintain, or bulk): ")
		fmt.Scanln(&u.Phase)

		// Validate user response.
		if u.Phase == "cut" || u.Phase == "maintain" || u.Phase == "bulk" {
			return
		}
		fmt.Println("Invalid diet phase. Please try again.")
	}
}

// promptDietOptions prints diet goal options, prompts for diet goal,
// and validates user response.
func promptDietOptions(u *UserInfo) string {
	var g string

	// Print to user recommended and custom diet goal options.
	fmt.Println("Recomended:")
	switch u.Phase {
	case "cut":
		fmt.Println("* Fat loss: 8 lbs in 8 weeks.")
	case "maintain":
		fmt.Println("* Maintenance: X weeks.")
	case "bulk":
		fmt.Println("* Muscle gain: X lbs in X weeks.")
	}
	fmt.Println("Custom: Choose diet duration and rate of weight change.")

	for {
		// Prompt user for diet goal.
		fmt.Printf("Enter diet goal (recommended or custom): ")
		fmt.Scanln(&g)

		// Validate user response.
		if g == "recommended" || g == "custom" {
			return g
		}
		fmt.Println("Invalid diet goal. Please try again.")
	}
}

func validateStartDate(u *UserInfo) {
	for {
		// Prompt user for diet start/stop date.
		fmt.Printf("Enter diet start date (YYYY-MM-DD) [Press Enter for today's date]: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		// If user entered default date
		if input == "" {
			input = time.Now().Format("2006-01-02")
		}

		// Validate user response.
		u.Phase.StartDate, err = time.Parse("2006-01-02", input)
		if err == nil {
			return
		}
		fmt.Println("Invalid date. Please try again.")
	}
}

func validateEndDate(u *UserInfo) {
	// TODO: Find min and max diet duration

	for {
		// Prompt user for diet start/stop date.
		fmt.Printf("Enter diet end date (YYYY-MM-DD): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		// TODO: Validate user response.
		// * Does end date fall after start date?
		// * Does end date fall under max diet duration?
		// * Does end date fall over min diet duration?
		//
		userInfo.Phase.EndDate, err = time.Parse("2006-01-02", input)
		if err == nil {
			return
		}
		fmt.Println("Invalid date. Please try again.")
	}
}

func promptDietGoal(u *UserInfo) {
	fmt.Println("Step 3: Choose diet goal.")

	// Prompt user for diet goal
	goal := promptDietOptions(u)

	// Fill out userInfo struct fields depending on whether user wants to
	// follow the recommended pace or custom pace.
	switch goal {
	case "recommended":
		reader := bufio.NewReader(os.Stdin)

		validateStartDate(u)

		// TODO: Find diet end date.
		// TODO: Find diet duration.
		// TODO: Calculate weekly weight change rate.

	case "custom":
		var w float64

		// Prompt user for goal weight
		fmt.Printf("Enter your goal weight: ")
		fmt.Scanln(&w)

		validateStartDate(u)
		validateEndDate(u)

		// TODO: Calculate diet duration
		// TODO: Calculate weekly weight change rate.

		// TODO: Find diet duration
	}
}

func promptConfirmation(u *UserInfo) {
	// Find difference from goal and current weight.
	diff := u.Phase.GoalWeight - u.Weight

	// Display current information to the user.
	fmt.Println("Summary:")
	fmt.Println("Diet duration: %s-%s (%f weeks)", u.Phase.StartDate, u.Phase.EndDate, diff)
	fmt.Printf("Target weight %f (%f)\n", u.Weight+diff, diff)
}

// validateGender prompts user for gender and validates their response.
func validateGender(u *UserInfo) {
	for {
		// Prompt user for gender.
		fmt.Print("Enter gender (male/female): ")
		fmt.Scanln(&u.Gender)

		// Validate user response.
		if u.Gender == "male" || u.Gender == "female" {
			return
		}
		fmt.Println("Must enter \"male\" or \"female\".")
	}
}

// validateWeight prompts user for weight and validates their
// response.
func validateWeight(u *UserInfo) {
	var weightStr string
	for {
		// Prompt user for weight.
		fmt.Print("Enter current weight: ")
		fmt.Scanln(&u.Weight)

		// Validate user response.
		w, err := strconv.ParseFloat(weightStr, 64)
		if err == nil && u.Weight > 0 {
			u.Weight = w
			return
		}
		fmt.Println("Invalid weight. Please try again.")
	}
}

// validateHeight prompts user for height and validates their response.
func validateHeight(u *UserInfo) {
	var heightStr string
	for {
		// Prompt user for height.
		fmt.Print("Enter height (cm): ")
		fmt.Scanln(&heightStr)

		// Validate user response.
		h, err := strconv.ParseFloat(weightStr, 64)
		if err == nil && h > 0 {
			u.Height = h
			return
		}
		fmt.Println("Invalid height. Please try again.")
	}
}

// validateAge prompts user for age and validates their response.
func validateAge(u *UserInfo) {
	var ageStr string
	for {
		// Prompt user for age.
		fmt.Print("Enter age: ")
		fmt.Scanln(&heightStr)

		// Validate user response.
		a, err := strconv.Atoi(ageStr)
		if err == nil && age > 0 {
			u.UserInfo = a
			return
		}
		fmt.Println("Invalid age. Please try again.")
	}
}

// validateActivity prompts user for activity level and validates their
// response.
func validateActivity(u *UserInfo) {
	for {
		// Prompt user for activity level.
		fmt.Print("Enter activity level (sedentary, light, moderate, active, very): ")
		fmt.Scanln(&u.ActivityLevel)

		// Validate user response.
		_, err := activity(u.ActivityLevel)
		if err == nil {
			return
		}
		fmt.Println("Invalid activity level. Please try again.")
	}
}

// promptUserInfo prompts for user details.
func promptUserInfo(u *UserInfo) {
	fmt.Println("Step 1: Your details.")
	validateGender(u)
	validateWeight(u)
	validateHeight(u)
	validateAge(u)
	validateActivity(u)
}

func ReadConfig() (*UserInfo, error) {
	var userInfo UserInfo

	// If the yaml config file doesn't exist
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		var userInfo UserInfo
		fmt.Println("Welcome! Please provide required information:")

		// Prompt for user information
		promptUserInfo(userInfo)
		promptDietType(userInfo)
		promptDietGoal(UserInfo)
		promptConfirmation(UserInfo)

		// Save user info to config file.
		err := saveUserInfo(userInfo)
		if err != nil {
			log.Println("Failed to save user info:", err)
			return nil, err
		}
		fmt.Println("User info saved successfully.")
	} else { // User has a config file.
		userInfo = UserInfo{}

		// Read YAML file.
		data, err := ioutil.ReadFile(configFilePath)
		if err != nil {
			log.Printf("Error: Can't read file: %v\n", err)
			return nil, err
		}

		// Unmarshal YAML data into struct.
		err = yaml.Unmarshal(data, &userInfo)
		if err != nil {
			log.Printf("Error: Can't unmarshal YAML: %v\n", err)
			return nil, err
		}
		fmt.Println("User info loaded successful.")
	}

	return userInfo, nil
}

// TODO
func Summary() {
	fmt.Println("Generating summary.")
	// Print day summary
	// Print week summary
	// Print month summary
}
