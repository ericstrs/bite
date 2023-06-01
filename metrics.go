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

const (
	CalsInProtein = 4 // Calories per gram of protein.
	CalsInCarbs   = 4 // Calories per gram of carbohydrate.
	CalsInFats    = 9 // Calories per gram of fat.
	weightCol     = 0 // Dataframe column for weight.
	calsCol       = 1 // Dataframe column for calories.
	dateCol       = 2 // Dataframe column for weight.
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
	TDEE          float64   `yaml:"tdee"`
	Macros        Macros    `taml:"macros"`
	Phase         PhaseInfo `yaml:"phase"`
}

type PhaseInfo struct {
	Name         string    `yaml:"name"`
	GoalCalories float64   `yaml:"goal_calories"`
	StartWeight  float64   `yaml:"start_weight"`
	GoalWeight   float64   `yaml:"goal_weight"`
	WeeklyChange float64   `yaml:"weekly_change"`
	StartDate    time.Time `yaml:"start_date"`
	EndDate      time.Time `yaml:"end_date"`
	Duration     float64   `yaml:"duration"`
	MaxDuration  float64   `yaml:"max_duration"`
	MinDuration  float64   `yaml:"min_duration"`
}

type Macros struct {
	Protein    float64 `yaml:"protein"`
	MinProtein float64 `yaml:"min_protein"`
	Carbs      float64 `yaml:"carbs"`
	MinCarbs   float64 `yaml:"min_carbs"`
	Fats       float64 `yaml:"fats"`
	MaxFats    float64 `yaml:"max_fats"`
	MinFats    float64 `yaml:"min_fats"`
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
	remaining := (protein * CalsInProtein) + (fats * CalsInFats)
	carbs := remaining / CalsInCarbs

	return protein, carbs, fats
}

// Metrics prints user TDEE, suggested macro split, and generates
// plots using logs data frame.
func Metrics(logs dataframe.DataFrame, u *UserInfo) {
	if logs.NRows() < 1 {
		log.Println("Error: Not enough entries to produce metrics.")
		return
	}

	// Get most recent weight as a string.
	vals := logs.Series[weightCol].Value(logs.NRows() - 1).(string)
	// Convert string to float64.
	weight, err := strconv.ParseFloat(vals, 64)
	if err != nil {
		log.Println("Failed to convert string to float64:", err)
		return
	}

	// Get BMR.
	bmr := Mifflin(weight, u.Height, u.Age, u.Gender)
	fmt.Printf("BMR: %.2f\n", bmr)

	// Get TDEE.
	t := TDEE(bmr, u.ActivityLevel)
	fmt.Printf("TDEE: %.2f\n", t)

	// Get suggested macro split.
	protein, carbs, fats := Macros(weight, 0.4)
	fmt.Printf("Protein: %.2fg Carbs: %.2fg Fats: %.2fg\n", protein, carbs, fats)

	// Create plots
}

// Save user information to yaml config file.
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
// response. This function is only used when the user picks the custom
// diet option.
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

			// Calculate daily deficit needed.
			deficit := u.Phase.WeeklyChange * 500
			// Set diet goal calories.
			u.Phase.GoalCalories = u.TDEE - deficit
		case "maintain":
			// Diet goal calories should be TDEE to maintain same weight.
			u.Phase.GoalCalories = u.TDEE
		case "bulk":
			// Ensue that goal weight is greater than starting weight.
			if g > u.Phase.StartWeight {
				u.Phase.GoalWeight = g
				return
			}

			// Calculate daily surplus needed.
			deficit := u.Phase.WeeklyChange * 500
			// Set diet goal calories.
			u.Phase.GoalCalories = u.TDEE + deficit
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

	// TODO: For both recommended and custom diets, calculate and set the
	// diet phase goal calories.
	//
	// What do we need to do? We already have TDEE. Now, we just need to

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

			// Calculate daily deficit needed.
			deficit := u.Phase.WeeklyChange * 500
			// Set diet goal calories.
			u.Phase.GoalCalories = u.TDEE - deficit
		case "maintain":
			u.Phase.WeeklyChange = 0
			u.Phase.Duration = 5
			// Set goal weight as diet starting weight.
			u.Phase.GoalWeight = u.Phase.StartWeight

			// Diet goal calories should be TDEE to maintain same weight.
			u.Phase.GoalCalories = u.TDEE
		case "bulk":
			u.Phase.WeeklyChange = 0.25
			u.Phase.Duration = 10

			// Calculate expected change in weight.
			a := u.Phase.StartWeight * u.Phase.WeeklyChange
			// Calculate goal weight.
			u.Phase.GoalWeight = u.Phase.StartWeight + a

			// Calculate daily surplus needed.
			deficit := u.Phase.WeeklyChange * 500
			// Set diet goal calories.
			u.Phase.GoalCalories = u.TDEE + deficit
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

	// Get BMR
	bmr := Mifflin(u.Weight, u.Height, u.Age, u.Gender)

	// Set TDEE
	u.TDEE = TDEE(bmr, u.ActivityLevel)
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

	// Set suggested macro split
	protein, carbs, fats := Macros(weight, 0.4)
	u.Macros.Protein = protein
	u.Macros.Carbs = carbs
	u.Macros.Fats = fats

	// Find min and max values for macros.
	setMinMaxMacros(u)

	// Print new phase information to user.
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

// setMinMaxMacros calculates the minimum and maximum macronutrient in
// grams using user's most recent logged bodyweight.
//
// Note: Min and max values are determined for general health. More
// active lifestyles will likely require different values.
func setMinMaxMacros(u *UserInfo) {
	// Minimum protein daily intake needed for health is about
	// 0.3g of protein per pound of bodyweight.
	u.Macros.ProteinMin = 0.3 * u.Weight
	// Maximum protein intake for general health is 2g of protein per
	// pound of bodyweight.
	u.Macros.ProteinMax = 2 * u.Weight

	// Minimum carb daily intake needed for *health* is about
	// 0.3g of carb per pound of bodyweight.
	u.Macros.CarbsMin = 0.3 * u.Weight
	// Maximum carb intake for general health is 5g of carb per pound of
	// bodyweight.
	u.Macros.CarbsMax = 5 * u.Weight

	// Minimum daily fat intake is 0.3g per pound of bodyweight.
	u.Macros.FatsMin = 0.3 * u.Weight
	// Maximum daily fat intake is keeping calorie contribtions from fat
	// to be under 40% of total daily calories .
	u.Macros.FatsMax = 0.4 * u.Phase.GoalCalories / CalsInFats
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
	t := time.Now() // Get current date.

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
		err := checkBulkGain(u, logs)
		if err != nil {
			return err
		}
	}

	return nil
}

// checkCutLoss checks to see if user is on the track to meeting weight
// loss goal.
func checkCutLoss(u *UserInfo, logs *dataframe.DataFrame) error {
	consecutiveMissedWeeks := 0

	// Iterate over each week of the diet.
	for date := u.Phase.StartDate; date.Before(u.Phase.EndDate); date = date.AddDate(0, 0, 7) {
		weekStart := date
		weekEnd := date.AddDate(0, 0, 6)

		result, err := metWeightLossGoal(logs, weekStart, weekEnd, u.Phase.WeeklyChange)
		if err != nil {
			return err
		}
		// If week has not met the weight loss goal, then restart the count.
		if result {
			consecutiveMissedWeeks = 0
			continue
		}

		consecutiveMissedWeeks++ // Update the count.

		// If there has been 2 weeks of the user not meeting the weekly
		// weight loss goal, then update accordingly.
		if consecutiveMissedWeeks >= 2 {
			fmt.Printf("The weekly weight loss goal of %f has not been met for two consecutive weeks.", u.Phase.WeeklyChange)
			adjustCutPhase(u) // Adjust weight loss plan.
		}
	}
	return nil
}

// findEntryIdx finds the index of an entry given a date.
func findEntryIdx(logs *dataframe.DataFrame, weekStart time.Time) (int, error) {
	// Find the index of weekStart in the data frame.
	for i := 0; i < logs.NRows(); i++ {
		date, err := time.Parse("2006-02-01", logs.Series[dateCol].Value(i).(string))
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

	return startIdx, nil
}

// metWeightLossGoal checks if the user has met the weekly weight loss
// given a single week.
func metWeightLossGoal(logs *dataframe.DataFrame, weekStart, weekEnd time.Time, weeklyChange float64) (bool, error) {
	var numDays int
	totalWeightChange := 0.0

	// Get the dataframe index of the entry with the start date of the
	// diet.
	startIdx := findEntryIdx(logs, weekStart)

	for i := 0; i < 7; i++ {
		// Get entry date.
		date, err := time.Parse("2006-02-01", logs.Series[dateCol].Value(i).(string))
		if err != nil {
			log.Println("ERROR: Couldn't parse date:", err)
			return nil, err
		}

		// Check if date is within the week.
		if date.After(weekEnd) {
			break
		}

		// Get entry weight.
		w := logs.Series[weightCol].Value(i).(string)
		weight, err := strconv.ParseFloat(w, 64)
		if err != nil {
			log.Println("ERROR: Failed to convert string to float64:", err)
			return nil, err
		}

		// Get entry calories.
		c := logs.Series[calsCol].Value(i).(string)
		cals, err := strconv.ParseFloat(c, 64)
		if err != nil {
			log.Println("ERROR: Failed to convert string to float64:", err)
			return nil, err
		}

		// If entry is the first in the dataframe, set previous weight
		// equal to zero. This is necessary to prevent index out of bounds
		// error.
		if startIdx == 0 {
			previousWeight := 0
		} else {
			// Otherwise, get use previous entry's date to get previous day
			// weight.

			// Get entry date.
			date, err := time.Parse("2006-02-01", logs.Series[dateCol].Value(i-1).(string))
			if err != nil {
				log.Println("ERROR: Couldn't parse date:", err)
				return nil, err
			}

			// If date is after the diet start date,
			if date.After(weekStart) {
				// Get previous entry's weight.
				pw := logs.Series[weightCol].Value(i - 1).(string)
				previousWeight, err := strconv.ParseFloat(c, 64)
				if err != nil {
					log.Println("ERROR: Failed to convert string to float64:", err)
					return nil, err
				}
			} else { // Previous entry's date is before the diet has started.
				previousWeight := 0
			}
		}

		// Caculate the weight change between two days.
		weightChange := weight - previousWeight
		// Update total weight change
		totalWeightChange += weightChange
	}

	// If there we zero entries found in the week, then return.
	if numDays == 0 {
		return false, nil
	}

	// Calculate the average change in weight over the week.
	averageWeightChange := totalWeightChange / float64(numDays)

	return averageWeightChange >= weeklyChange, nil
}

// adjustCutPhase calulates the daily caloric deficit and then attempts
// to apply that deficit though first cutting fats, then carbs, and
// finally protein.
//
// The deficit will be applied up to the minimmum macro values.
func adjustCutPhase(u *UserInfo) {
	// Calculate the needed daily deficit.
	deficit := u.Phase.WeeklyChange * 500

	// Set cut calorie goal.
	u.Phase.GoalCalories = u.TDEE - deficit
	fmt.Print("Reducing caloric deficit by %f calories\n", deficit)

	// Convert caloric deficit to fats in grams.
	fatDeficit := deficit * CalsInFats

	// If the fat deficit is greater than or equal to the available fats
	// left (up to minimum fats limit), then apply the defict
	// exclusively through removing fats.
	if fatDeficit >= (u.Macros.Fats - u.Macros.MinFats) {
		u.Macros.Fats -= u.Macros.MinFats
		return
	} else { // Otherwise, there are not enough fats to competely apply the deficit, so remove the remaining fats.
		// Remove fats up to the minimum fats limit.
		remainingInFats := u.Macros.MinFats - deficit
		u.Macros.Fats -= deficit - remainingInFats
	}

	// Convert the remaining fats in grams to calories.
	remainingInCals := remainingInFats * CalsInFats
	// Convert the remaining calories to carbs in grams.
	carbDeficit := remainingInCals / CalsInCarbs

	// If carb deficit is greater than or equal to the availiable carbs
	// left (up to minimum carbs limit), then apply the remaining
	// deficit by removing carbs.
	if carbDeficit >= (u.Macros.Carbs - u.Macros.MinCarbs) {
		u.Macros.Carbs -= u.Macros.MinCarbs
		return
	} else {
		// Remove fats up to the minimum carbs limit.
		remainingInCarbs := u.Macros.MinCarbs - carbDeficit
		u.Macros.Carbs -= carbDeficit - remainingInCarbs
	}

	// Set protein deficit in grams to the carbs that could not be
	// removed.
	// Note: CalsInCarbs = CalsInProtein.
	proteinDeficit := remainingInCals

	// If protein deficit is greater than or equal to the availiable
	// protein left (up to minimum protein limit), then apply the
	// remaining deficit by removing protein.
	if proteinDeficit >= (u.Macros.Protein - u.Macros.MinProtein) {
		u.Macros.Protein -= u.Macros.MinProtein
		return
	} else {
		// Remove protein up to the minimum protein limit.
		remainingInProtein := u.Macros.MinProtein - proteinDeficit
		u.Macros.Protein -= proteinDeficit - remainingInProtein
	}

	// Convert the remaining protein in grams to calories.
	remaining := remainingInProtein * CalsInProtein

	// If the remaining calories are not zero, then stop removing macros
	// and update the diet goal calories.
	if remaining != 0 {
		fmt.Printf("Could not reach a deficit of %f as the minimum fat, carb, and protein values have been met.\n", deficit)
		fmt.Printf("Updating caloric deficit to %f\n", deficit-remaining)
		// Override initial cut calorie goal.
		u.Phase.GoalCalories = u.TDEE - deficit + remaining
	}
}

func checkBulkGain(u *UserInfo, logs *dataframe.DataFrame) error {
	consecutiveMissedWeeks := 0
	// If there has been 2 weeks of the user not meeting the weekly
	// weight gain goal, then update accordingly.
	for date := u.Phase.StartDate; date.Before(u.Phase.EndDate); date = date.AddDate(0, 0, 7) {
		weekStart := date
		weekEnd := date.AddDate(0, 0, 6)

		if metWeightGainGoal(logs, weekStart, weekEnd, u.Phase.WeeklyChange) {
			consecutiveMissedWeeks = 0
			continue
		}

		consecutiveMissedWeeks++

		if consecutiveMissedWeeks >= 2 {
			fmt.Printf("The weekly weight gain goal of %f has not been met for two consecutive weeks.", u.Phase.WeeklyChange)
			// Adjust weight loss plan.
			adjustBulkPhase(u)
		}
	}
	return nil
}

// adjustCutPhase calculates the caloric surplus and then attempts to
// apply it by first adding carbs, then fats, and finally fats.
func adjustBulkPhase(u *UserInfo) {
	// Calculate the needed daily surplus.
	surplus := u.Phase.WeeklyChange * 500

	// Set bulk calorie goal.
	u.Phase.GoalCalories = u.TDEE + surplus
	fmt.Print("Modifying caloric surplus by %f calories\n", surplus)

	// Convert surplus in calories to carbs in grams.
	carbSurplus := surplus * CalsInCarbs

	// If carb surplus is less than or equal to the availiable carbs
	// left (up to maximum carbs limit), then apply the surplus
	// execlusively through adding carbs and return.
	if u.Macros.MaxCarbs <= (u.Macros.Carbs + carbSurplus) {
		u.Macros.Carbs += carbSurplus
		return
	}
	// Otherwise, there are too many carbs to completely apply the surplus.

	// Calculate how many grams of carbs are over the maximum carb limit.
	remainingInCarbs := (u.Macros.Carbs + carbSurplus) - u.Macros.MaxCarbs
	// Add carbs up the maximum carb limit.
	u.Macros.Carbs += carbSurplus - remainingInCarbs

	// Convert the carbs in grams that couldn't be included to calories.
	remainingInCals := remainingInCarbs * CalsInCarbs
	// Convert the remaining surplus from calories to fats.
	fatSurplus := remainingInCals / CalsInFats

	// If the fat deficit is less than or equal to the available fats
	// left (up to maximum fats limit), then apply the surplus
	// exclusively through adding fats and return.
	if u.Macros.MaxFats <= (u.Macros.Fats + fatSurplus) {
		u.Macros.Fats += fatSurplus
		return
	}
	// Otherwise, there are too many fats to competely apply the surplus.

	// Calculate how many grams of fats are over the maximum fat limit.
	remainingInFats := (u.Macros.Fats + fatSurplus) - u.Macros.MaxFats
	// Add fats up the maximum fat limit.
	u.Macros.Fats += fatSurplus - remainingInFats

	// Convert the fats in grams that couldn't be included to calories.
	remainingInCals = remainingInFats * CalsInFats
	// Convert the remaining surplus from calories to protein.
	proteinSurplus := remainingInCals / CalsInProtein

	// If protein surplus is less than or equal to the availiable
	// protein left (up to maximum protein limit), then apply the
	// surplus by adding protein and return.
	if u.Macros.MaxProtein <= (u.Macros.Protein + proteinSurplus) {
		u.Macros.Protein += proteinSurplus
		return
	}
	// Otherwise, there is too much protein to completely apply the
	// surplus.

	// Calculate how many grams of protein are over the maximum protein limit.
	remainingInProtein := u.Macros.Protein + proteinSurplus - u.Macros.MaxProtein
	// Add protein up to the maximum protein limit.
	u.Macros.Protein += proteinSurplus - remainingInProtein

	// Convert the protein in grams that couldn't be included to calories.
	remaining := remainingInProtein * CalsInProtein

	// If the remaining calories are not zero, then stop adding to macros
	// and update the diet goal calories and return.
	if remaining != 0 {
		fmt.Printf("Could not reach a surplus of %f as the minimum fat, carb, and protein values have been    met.\n", surplus)
		fmt.Printf("Updating caloric surplus to %f\n", surplus-remaining)
		// Override initial cut calorie goal.
		u.Phase.GoalCalories = u.TDEE + surplus - remaining
		return
	}
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

	startIdx, err := findEntryIdx(logs, weekStart)
	if err != nil {
		return 0, err
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
