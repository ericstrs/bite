// Calories package provides dietary insight.
package calories

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rocketlaunchr/dataframe-go"
	"gopkg.in/yaml.v2"
)

/*
const ConfigFilePath = "./config.yaml"
const EntriesFilePath = "./data.csv"
*/

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
	StartWeight  float64   `yaml:"start_weight"`
	GoalWeight   float64   `yaml:"goal_weight"`
	WeeklyChange float64   `yaml:"weekly_change"`
	StartDate    time.Time `yaml:"start_date"`
	EndDate      time.Time `yaml:"end_date"`
	Duration     float64   `yaml:"duration"`
	MaxDuration  float64   `yaml:"max_duration"`
	MinDuration  float64   `yaml:"min_duration"`
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

// Metrics prints user BMR, TDEE, suggested macro split, and generates
// plots using logs data frame.
func Metrics(logs dataframe.DataFrame, userInfo UserInfo) {
	if logs.NRows() < 1 {
		log.Println("Error: Not enough entries to produce metrics.")
		return
	}

	// Get most recent weight as a string.
	vals := logs.Series[0].Value(logs.NRows() - 1).(string)
	// Convert string to float64.
	weight, err := strconv.ParseFloat(vals, 64)
	if err != nil {
		log.Println("Failed to convert string to float64:", err)
		return
	}

	// Get BMR
	bmr := Mifflin(weight, userInfo.Height, userInfo.Age, userInfo.Gender)
	fmt.Printf("BMR: %.2f\n", bmr)

	// Get TDEE
	t := TDEE(bmr, userInfo.ActivityLevel)
	fmt.Printf("TDEE: %.2f\n", t)

	// Get suggested macro split
	protein, carbs, fats := Macros(weight, 0.4)
	fmt.Printf("Protein: %.2fg Carbs: %.2fg Fats: %.2fg\n", protein, carbs, fats)

	// Create plots
}

// Save userInfo to yaml config file.
func saveUserInfo(u *UserInfo) error {
	data, err := yaml.Marshal(u)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(ConfigFilePath, data, 0644)
}

// promptUserType prompts the user to enter desired diet phase.
func promptDietType(u *UserInfo) {
	// Prompt user for diet type
	fmt.Println("Step 2: Choose diet type.")

	fmt.Println("Fat loss (cut). Lose fat while losing weight and preserving muscle.")
	fmt.Println("Maintenance (maintain). Stay at your current weight.")
	fmt.Println("Muscle gain (bulk). Gain muscle while minimizing fat.")

	var s string
	for {
		// Prompt user for diet phase.
		fmt.Print("Enter phase (cut, maintain, or bulk): ")
		fmt.Scanln(&s)

		// Validate user response.
		if s == "cut" || s == "maintain" || s == "bulk" {
			u.Phase.Name = s
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
	fmt.Printf("Recommended: ")
	switch u.Phase.Name {
	case "cut":
		fmt.Printf("Lose 4 lbs in 8 weeks.\n")
	case "maintain":
		fmt.Printf("Maintain same weight for 5 weeks.\n")
	case "bulk":
		fmt.Printf("Gain 2.5 lbs in 10 weeks.\n")
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

// validateStartDate prompts user for diet start date and validates user
// response.
func validateStartDate(u *UserInfo) {
	reader := bufio.NewReader(os.Stdin)
	for {
		// Prompt user for diet start date.
		fmt.Printf("Enter diet start date (YYYY-MM-DD) [Press Enter for today's date]: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		// If user entered default date
		if input == "" {
			input = time.Now().Format("2006-01-02")
		}

		// Validate user response.
		d, err := time.Parse("2006-01-02", input)
		if err == nil {
			u.Phase.StartDate = d
			return
		}
		fmt.Println("Invalid date. Please try again.")
	}
}

// validateEndDate prompts user for diet end date and validates user
// response.
func validateEndDate(u *UserInfo) {
	reader := bufio.NewReader(os.Stdin)
	for {
		// Prompt user for diet stop date.
		fmt.Printf("Enter diet end date (YYYY-MM-DD): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		d, err := time.Parse("2006-01-02", input)
		// Calculate diet duration given current end date.
		dur := calculateDuration(u.Phase.StartDate, d).Hours() / 24 / 7

		// Validate user response:
		// * Does end date fall after start date?
		// * Is diet duration less than max diet duration?
		// * Is diet duration greater than min diet duration?
		//
		fmt.Printf("Duration: %f. MaxDuration: %f. MinDuration: %f\n", dur, u.Phase.MaxDuration, u.Phase.MinDuration)
		if err == nil && d.After(u.Phase.StartDate) && dur < u.Phase.MaxDuration && dur > u.Phase.MinDuration {
			u.Phase.EndDate = d
			u.Phase.Duration = dur
			return
		}
		fmt.Println("Invalid date. Please try again.")
	}
}

// calculateEndDate calculates the diet end date given diet start date
// and diet duration in weeks.
func calculateEndDate(d time.Time, duration float64) time.Time {
	endDate := d.AddDate(0, 0, int(duration*7.0))
	return endDate
}

// calculateDuration calculates and returns diet duration in weeks given
// start and end date.
func calculateDuration(start, end time.Time) time.Duration {
	d := end.Sub(start)
	return d
}

// calculateWeeklyChange calculates and returns the weekly weight
// change given current weight, goal weight, and diet duration.
func calculateWeeklyChange(current, goal, duration float64) float64 {
	weeklyChange := (current - goal) / duration
	return weeklyChange
}

// validateGoalWeight prompts user for goal weight and validates their
// response.
//
// If phase is maintenance:
// * ensure goal weight is within +/- 1.25 current weight.
//
// If phase is cut:
// *  cut goal should be smaller than current weight.
//
// If phase is bulk:
// * bulk goal should be bigger than current weight.
//
// And check min max weight bounds for a single cut/bulk.
func validateGoalWeight(u *UserInfo) {
	if u.Phase.Name == "maintain" {
		u.Phase.GoalWeight = u.Phase.StartWeight
		return
	}

	var g float64
	for {
		// Prompt user for goal weight.
		fmt.Printf("Enter your goal weight: ")
		fmt.Scanln(&g)

		switch u.Phase.Name {
		case "cut":
			// Ensure goal weight doesn't exceed 10% of starting body weight.
			lowerBound := u.Phase.StartWeight * 0.10
			if g > u.Phase.StartWeight-lowerBound {
				u.Phase.GoalWeight = g
				return
			}
		case "maintain":
		case "bulk":
			// Ensue that goal weight is greater than starting weight.
			if g > u.Phase.StartWeight {
				u.Phase.GoalWeight = g
				return
			}
		}
		fmt.Println("Invalid goal weight. Please try again.")
	}
}

// promptDietGoal prompts the user diet goal and diet details.
func promptDietGoal(u *UserInfo) {
	fmt.Println("Step 3: Choose diet goal.")

	// Prompt user for diet goal
	goal := promptDietOptions(u)

	// Set min and max diet durations.
	switch u.Phase.Name {
	case "cut":
		u.Phase.MaxDuration = 12
		u.Phase.MinDuration = 6
	case "maintain":
		u.Phase.MaxDuration = math.Inf(1)
		u.Phase.MinDuration = 4
	case "bulk":
		u.Phase.MaxDuration = 16
		u.Phase.MinDuration = 6
	}

	// Fill out remaining userInfo struct fields given user preference on
	// recommended or custom diet pace.
	switch goal {
	case "recommended":
		validateStartDate(u)

		switch u.Phase.Name {
		case "cut":
			u.Phase.WeeklyChange = 1
			u.Phase.Duration = 8

			// Calculate expected change in weight.
			a := u.Phase.StartWeight * u.Phase.WeeklyChange
			// Calculate goal weight.
			u.Phase.GoalWeight = u.Phase.StartWeight - a
		case "maintain":
			u.Phase.WeeklyChange = 0
			u.Phase.Duration = 5
			// Set goal weight as diet starting weight.
			u.Phase.GoalWeight = u.Phase.StartWeight
		case "bulk":
			u.Phase.WeeklyChange = 0.25
			u.Phase.Duration = 10

			// Calculate expected change in weight.
			a := u.Phase.StartWeight * u.Phase.WeeklyChange
			// Calculate goal weight.
			u.Phase.GoalWeight = u.Phase.StartWeight + a
		}

		// Calculate the diet end date.
		u.Phase.EndDate = calculateEndDate(u.Phase.StartDate, u.Phase.Duration)
	case "custom":
		// Prompt and validate user diet goal weight.
		validateGoalWeight(u)

		// Prompt and validate user diet start date.
		validateStartDate(u)

		// Prompt and validate user diet end date.
		validateEndDate(u)

		// Calculate weekly weight change rate.
		u.Phase.WeeklyChange = calculateWeeklyChange(u.Weight, u.Phase.GoalWeight, u.Phase.Duration)
	}
}

// promptConfirmation prints diet summary to the user.
func promptConfirmation(u *UserInfo) {
	// Find difference from goal and start weight.
	diff := u.Phase.GoalWeight - u.Phase.StartWeight

	// Display current information to the user.
	fmt.Println("Summary:")
	fmt.Printf("Diet duration: %s-%s (%f weeks)\n", u.Phase.StartDate, u.Phase.EndDate, u.Phase.Duration)

	switch u.Phase.Name {
	case "cut":
		fmt.Printf("Target weight %f (%.2f lbs)\n", u.Weight+diff, diff)
		fmt.Println("During your cut, you should lean slightly on the side of doing more high-volume training.")
	case "maintain":
		fmt.Printf("Target weight %f (+%.2f lbs)\n", u.Weight+diff, diff)
		fmt.Println("During your maintenance, you should lean towards low-volume training (3-10 rep strength training). Get active rest (barely any training and just living life for two weeks is also an option). This phase is meant to give your body a break to recharge for future hard training.")
	case "bulk":
		fmt.Printf("Target weight %f (+%.2f lbs)\n", u.Weight+diff, diff)
		fmt.Println("During your bulk, you can just train as you normally would.")
	}
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
		fmt.Scanln(&weightStr)

		// Validate user response.
		w, err := strconv.ParseFloat(weightStr, 64)
		if err == nil && w > 0 {
			u.Weight = w
			u.Phase.StartWeight = w
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
		h, err := strconv.ParseFloat(heightStr, 64)
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
		fmt.Scanln(&ageStr)

		// Validate user response.
		a, err := strconv.Atoi(ageStr)
		if err == nil && a > 0 {
			u.Age = a
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

// ReadConfig created config file if it doesn't exist or reads in
// existing config file and returns userInfo.
func ReadConfig() (*UserInfo, error) {
	var u UserInfo

	// If the yaml config file doesn't exist
	if _, err := os.Stat(ConfigFilePath); os.IsNotExist(err) {
		fmt.Println("Welcome! Please provide required information:")

		// Prompt for user information
		promptUserInfo(&u)
		promptDietType(&u)
		promptDietGoal(&u)
		promptConfirmation(&u)

		// Save user info to config file.
		err := saveUserInfo(&u)
		if err != nil {
			log.Println("Failed to save user info:", err)
			return nil, err
		}
		fmt.Println("User info saved successfully.")
	} else { // User has a config file.
		u = UserInfo{}

		// Read YAML file.
		data, err := ioutil.ReadFile(ConfigFilePath)
		if err != nil {
			log.Printf("Error: Can't read file: %v\n", err)
			return nil, err
		}

		// Unmarshal YAML data into struct.
		err = yaml.Unmarshal(data, &u)
		if err != nil {
			log.Printf("Error: Can't unmarshal YAML: %v\n", err)
			return nil, err
		}
		fmt.Println("User info loaded successful.")
	}

	return &u, nil
}

// promptTransition prints a diet recap and suggested next diet phase,
// prompts them to choose the next phase, and saves next phase to
// config file.
func promptTransition(u *UserInfo) error {
	s := u.Phase.Name
	fmt.Printf("%s phase completed. Beginning diet transition.", strings.ToUpper(s[:1])+s[1:])
	fmt.Println("Step 1: Diet phase recap")
	fmt.Printf("Goal weight: %f. Current weight: %f\n", u.Phase.GoalWeight, u.Weight)

	switch u.Phase.Name {
	case "cut":
		// Print suggestion to start a maintenance period of the same
		// duration as the completed cut.
		fmt.Println("A maintenance period of the same duration as your completed cut is recommended.")
	case "maintain":
	case "bulk":
		// Print suggestion to begin a maintenance phase
		fmt.Println("A maintenance period of the at least a month is recommended.")

		// TODO: If the user does follow recommendation of a maintenance
		// phase, then
		// 1. Calculate your weekly calorie surplus based on the average weight
		// gain over the final two weeks of bulking.
		// 2. Start out phase by decreasing your caloric intake by that amount.
	}

	// Prompt user to start a new diet phase
	promptDietType(u)
	promptDietGoal(u)
	promptConfirmation(u)

	// Save user info to config file.
	err := saveUserInfo(u)
	if err != nil {
		log.Println("Failed to save user info:", err)
		return err
	}
	fmt.Println("User info saved successfully.")

	return nil
}

// CheckProgress performs checks on the user's current diet phase.
//
// Current solution to defining a week is continually adding 7 days to
// the start date. We consider weeks where the user has a consistency of
// adding at least two entries for a given week.
//
// Note: Converting duration in weeks (float64) to int is taking the
// floor which may truncate days and this may lead to some issues.
func CheckProgress(u *UserInfo, logs *dataframe.DataFrame) error {
	// Get current date.
	t := time.Now()

	// If today comes before diet start date, then phase has not yet begun.
	if t.Before(u.Phase.StartDate) {
		return nil
	}
	// If today comes after diet end date, diet phase is over.
	if t.After(u.Phase.EndDate) {
		// Prompt for phase transition
		err := promptTransition(u)
		if err != nil {
			return err
		}

		return nil
	}

	// Make a map to track the numbers of entries in each week.
	entryCountPerWeek := make(map[int]int)

	i := 0
	// Iterate over weeks within the diet phase.
	for date := u.Phase.StartDate; date.Before(u.Phase.EndDate); date = date.AddDate(0, 0, 7) {
		weekStart := date
		weekEnd := date.AddDate(0, 0, 6)

		// Count the number of entries within the current week.
		entryCount, err := countEntriesInWeek(logs, weekStart, weekEnd)
		if err != nil {
			return err
		}

		weekNumber := i
		i++
		entryCountPerWeek[weekNumber] = entryCount
	}

	count := 0
	// If there is less than 2 weeks of entries after the diet start date,
	// then do nothing, and return
	for week := 1; week <= int(u.Phase.Duration); week++ {
		if entryCountPerWeek[week] > 2 {
			count++
		}
	}
	if count < 2 {
		log.Println("There is less than 2 weeks of entries after the diet start date. Skipping remaining checks on user progress.")
		return nil
	}

	switch u.Phase.Name {
	case "cut":
		// Ensure user has not lost too much weight.
		err := checkCutThreshold(u)
		if err != nil {
			return err
		}

		// Ensure user is meeting weekly weight loss.
		err = checkCutLoss(u, logs)
		if err != nil {
			return err
		}
	case "maintain":
	case "bulk":
	}

	return nil
}

func checkCutLoss(u *UserInfo, logs *dataframe.DataFrame) error {
	consecutiveMissedWeeks := 0
	// If there has been 2 weeks of the user not meeting the weekly
	// weight loss goal, then update accordingly.
	for date := u.Phase.StartDate; date.Before(u.Phase.EndDate); date = date.AddDate(0, 0, 7) {
		weekStart := date
		weekEnd := date.AddDate(0, 0, 6)

		if metWeightLossGoal(logs, weekStart, weekEnd, u.Phase.WeeklyChange) {
			consecutiveMissedWeeks = 0
			continue
		}

		consecutiveMissedWeeks++

		if consecutiveMissedWeeks >= 2 {
			fmt.Printf("The weekly weight loss goal of %f has not been met for two consecutive weeks.")
			// TODO: Call function to adjust weight loss plan.
		}
	}
	return nil
}

// checkCutThreshold checks if the user has lost too much weight, in
// which the cut is stopped and a maintenance phase begins.
//
// Note: diet duration is left unmodified so maintenance phase
// lasts as long as the cut.
func checkCutThreshold(u *UserInfo) error {
	// Find the amount of weight the user has lost.
	weightLost := u.Phase.StartWeight - u.Phase.Weight
	// Find the theshold weight the user is allowed to lose.
	threshold := u.Phase.StartWeight * 0.10

	// If the user has lost more than 10% of starting weight,
	if weightLost > threshold {
		fmt.Println("Warning: You've reached the maximum threshold for weight loss (you've lost more than 10% of your starting weight in a single cutting phase). Stopping your cut and beginning a maintenance phase.")

		// Stop cut phase and set phase to maintenance.
		u.Phase.Name = "maintain"
		// Immediately start maintenance phase.
		u.Phase.StartDate = time.Now().Format("2006-01-02")
		u.Phase.WeeklyChange = 0
		u.Phase.GoalWeight = u.Phase.StartWeight
		// Calculate the diet end date.
		u.Phase.EndDate = calculateEndDate(u.Phase.StartDate, u.Phase.Duration)

		promptConfirmation(u)

		// Save user info to config file.
		err := saveUserInfo(u)
		if err != nil {
			log.Println("Failed to update phase to maintenance:", err)
			return err
		}
		fmt.Println("Phase successfully updated to maintenance.")

		return nil
	}
}

// countEntriesInWeek finds the number of entires within a given week.
func countEntriesInWeek(logs *dataframe.DataFrame, weekStart, weekEnd time.Time) (int, error) {
	count := 0

	startIdx := -1
	// Find the index of weekStart in the data frame.
	for i := 0; i < logs.NRows(); i++ {
		date, err := time.Parse("2006-02-01", logs.Series[2].Value(i).(string))
		if err != nil {
			log.Println("ERROR: Couldn't parse date:", err)
			return 0, err
		}
		if date.Before(weekStart) {
			continue
		}
		startIdx = i
		break
	}

	// If start date has passed,
	if startIdx != -1 {
		// Starting from the start date index, iterate over the week, and
		// update counter when an entry is encountered.
		for i := startIdx; i < logs.NRows(); i++ {
			date, err := time.Parse("2006-02-01", logs.Series[2].Value(i).(string))
			if err != nil {
				log.Println("ERROR: Couldn't parse date:", err)
				return 0, err
			}
			if date.Before(weekStart) || date.After(weekEnd) {
				break
			}
			count++
		}
	}

	return count, nil
}

// TODO
func Summary() {
	fmt.Println("Generating summary.")
	// Print day summary
	// Print week summary
	// Print month summary
}
