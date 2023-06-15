// Calories package provides dietary insight.
package calories

import (
	"bufio"
	"errors"
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

type WeightGainStatus int
type WeightLossStatus int
type WeightMaintenanceStatus int

const (
	calsPerPound                            = 3500 // Estimated calories per pound of bodyweight.
	gainedTooLittle WeightGainStatus        = -1
	withinGainRange WeightGainStatus        = 0
	gainedTooMuch   WeightGainStatus        = 1
	lostTooLittle   WeightLossStatus        = -1
	withinLossRange WeightLossStatus        = 0
	lostTooMuch     WeightLossStatus        = 1
	lost            WeightMaintenanceStatus = -1
	maintained      WeightMaintenanceStatus = 0
	gained          WeightMaintenanceStatus = 1
)

type PhaseInfo struct {
	Name            string    `yaml:"name"`
	GoalCalories    float64   `yaml:"goal_calories"`
	StartWeight     float64   `yaml:"start_weight"`
	GoalWeight      float64   `yaml:"goal_weight"`
	WeeklyChange    float64   `yaml:"weekly_change"`
	StartDate       time.Time `yaml:"start_date"`
	EndDate         time.Time `yaml:"end_date"`
	LastCheckedDate time.Time `yaml:"last_checked_date"`
	Duration        float64   `yaml:"duration"`
	MaxDuration     float64   `yaml:"max_duration"`
	MinDuration     float64   `yaml:"min_duration"`
	Active          bool      `yaml:"active"`
}

// ReadConfig reads config file or creates it if it doesn't exist and
// returns UserInfo struct.
func ReadConfig() (u *UserInfo, err error) {
	// If no config file exists,
	if _, err := os.Stat(ConfigFilePath); os.IsNotExist(err) {
		u, err = generateConfig()
		if err != nil {
			return nil, err
		}

		return u, nil
	}
	// Otherwise, user has a config file.

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
	log.Println("Loaded user info.")

	return u, nil
}

// generateConfig generates a new config file for the user.
func generateConfig() (*UserInfo, error) {
	fmt.Println("Welcome! Please provide required information:")

	u := UserInfo{}

	// Get user details.
	getUserInfo(&u)

	processUserInfo(&u)

	// Save user info to config file.
	err := saveUserInfo(&u)
	if err != nil {
		log.Println("Failed to save user info:", err)
		return nil, err
	}

	return &u, nil
}

// getUserInfo prompts for user details.
func getUserInfo(u *UserInfo) {
	fmt.Println("Step 1: Your details.")
	u.Sex = getSex()

	w := getWeight()
	u.Weight = w

	u.Height = getHeight()
	u.Age = getAge()
	u.ActivityLevel = getActivity()

	// Get BMR
	bmr := Mifflin(u)

	// Set TDEE
	u.TDEE = TDEE(bmr, u.ActivityLevel)
}

// getSex prompts user for their sex, validates their response, and
// returns valid sex.
func getSex() (s string) {
	for {
		// Prompt user for their sex.
		s = promptSex()

		// Validate user response.
		err := validateSex(s)
		if err != nil {
			fmt.Println("Must enter \"male\" or \"female\". Please try again.")
			continue
		}

		break
	}

	return s
}

// promptSex prompts and returns user sex.
func promptSex() (s string) {
	// Prompt user for their sex.
	fmt.Print("Enter sex (male/female): ")
	fmt.Scanln(&s)
	return s
}

// validateSex validates user sex and returns sex if valid.
func validateSex(s string) error {
	if s == "male" || s == "female" {
		return nil
	}
	return errors.New("Invalid sex.")
}

// getWeight prompts user for weight, validate their response, and
// returns valid weight.
func getWeight() (weight float64) {
	for {
		// Prompt user to enter weight.
		weightStr := promptWeight()

		var err error
		// Validate user response.
		weight, err = validateWeight(weightStr)
		if err != nil {
			fmt.Println("Invalid weight. Please try again.")
			continue
		}

		break
	}

	return weight
}

// promptWeight prompts the user to enter their weight.
func promptWeight() (weight string) {
	fmt.Print("Enter weight in lbs: ")
	fmt.Scanln(&weight)
	return weight
}

// validateWeight validates user response to being
// prompted for weight and returns conversion to float64 if valid.
func validateWeight(weightStr string) (w float64, err error) {
	w, err = strconv.ParseFloat(weightStr, 64)
	if err != nil || w < 0 {
		return 0, errors.New("Invalid weight.")
	}
	return w, nil
}

// getHeight prompts user for height, validates their response, and
// returns valid height.
func getHeight() (height float64) {
	for {
		// Prompt user for height.
		heightStr := promptHeight()

		var err error
		// Validate their response.
		height, err = validateHeight(heightStr)
		if err != nil {
			fmt.Println("Invalid height. Please try again.")
			continue
		}

		break
	}

	return height
}

// promptHeight prompts and returns user height as a string.
func promptHeight() (heightStr string) {
	fmt.Print("Enter height (cm): ")
	fmt.Scanln(&heightStr)
	return heightStr
}

// validateHeight validates user height and returns converion to from
// string to float64 if valid.
func validateHeight(heightStr string) (float64, error) {
	h, err := strconv.ParseFloat(heightStr, 64)
	if err != nil || h < 0 {
		return 0, errors.New("Invalid height.")
	}
	return h, nil
}

// getAge prompts user for age, validates their response, and returns
// valid age.
func getAge() (age int) {
	var err error
	for {
		// Prompt user for age.
		ageStr := promptAge()

		// Validate user response.
		age, err = validateAge(ageStr)
		if err != nil {
			fmt.Println("Invalid age. Please try again.")
			continue
		}

		break
	}
	return age
}

// promptAge prompts user for their age and returns age as a string.
func promptAge() (ageStr string) {
	fmt.Print("Enter age: ")
	fmt.Scanln(&ageStr)
	return ageStr
}

// validateAge validates user age and returns conversion from string to
// int if valid.
func validateAge(ageStr string) (int, error) {
	// Validate user response.
	a, err := strconv.Atoi(ageStr)
	if err != nil || a < 0 {
		return 0, errors.New("Invalid age.")
	}
	return a, nil
}

// getActivity prompts user for activity level, validates user
// response, and returns valid activity level.
func getActivity() (a string) {
	for {
		// Prompt user for activity.
		a = promptActivity()

		// Validate user response.
		err := validateActivity(a)
		if err != nil {
			fmt.Println("Invalid activity level. Please try again.")
			continue
		}

		break
	}
	return a
}

// promptActivity prompts and returns user activity level.
func promptActivity() (a string) {
	fmt.Print("Enter activity level (sedentary, light, moderate, active, very): ")
	fmt.Scanln(&a)
	return a
}

// validateActivity and validates their  response.
func validateActivity(a string) error {
	_, err := activity(a)
	if err != nil {
		return err
	}
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
	// Make a map to track the numbers of entries in each week.
	entryCountPerWeek, err := countEntriesPerWeek(u, logs)
	if err != nil {
		return err
	}

	// Count number of valid weeks.
	count := countValidWeeks(*entryCountPerWeek)

	// If there is less than 2 weeks of entries after the diet start date,
	// then do nothing, and return early.
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

		var total float64
		var status WeightLossStatus

		// Ensure user is meeting weekly weight loss.
		status, total, err = checkCutLoss(u, logs)
		if err != nil {
			return err
		}

		switch status {
		case lostTooLittle:
			fmt.Printf("The weekly weight gain goal of %f has not been met for two consecutive weeks.", u.Phase.WeeklyChange)
			addCals(u, total)
		case lostTooMuch:
			fmt.Printf("The weekly weight gain goal of %f has not been met for two consecutive weeks.", u.Phase.WeeklyChange)
			removeCals(u, total)
		case withinLossRange: // Do nothing
		}
	case "maintain":
		// Ensure user is maintaing weight.
		status, total, err := checkMaintenance(u, logs)
		if err != nil {
			return err
		}

		switch status {
		case lost:
			fmt.Printf("The weekly weight gain goal of %f has not been met for two consecutive weeks.", u.Phase.WeeklyChange)
			addCals(u, total)
		case gained:
			fmt.Printf("The weekly weight gain goal of %f has not been met for two consecutive weeks.", u.Phase.WeeklyChange)
			removeCals(u, total)
		case maintained: // Do nothing
		}
	case "bulk":
		// Ensure user has not gained too much weight.
		err := checkBulkThreshold(u)
		if err != nil {
			return err
		}

		var total float64
		var status WeightGainStatus
		// Ensure user is metting weekly weight gain.
		status, total, err = checkBulkGain(u, logs)
		if err != nil {
			return err
		}

		switch status {
		case gainedTooLittle:
			fmt.Printf("The weekly weight gain goal of %f has not been met for two consecutive weeks.", u.Phase.WeeklyChange)
			addCals(u, total)
		case gainedTooMuch:
			fmt.Printf("The weekly weight gain goal of %f has not been met for two consecutive weeks.", u.Phase.WeeklyChange)
			removeCals(u, total)
		case withinGainRange: // Do nothing
		}
	}

	return nil
}

// countEntriesPerWeek returns a map to tracker the number of entires in
// each weeks of a diet phase.
func countEntriesPerWeek(u *UserInfo, logs *dataframe.DataFrame) (*map[int]int, error) {
	entryCountPerWeek := make(map[int]int)

	weekNumber := 0
	// Iterate over weeks within the diet phase.
	for date := u.Phase.StartDate; date.Before(u.Phase.EndDate); date = date.AddDate(0, 0, 7) {
		weekStart := date
		weekEnd := date.AddDate(0, 0, 6)

		// Count the number of entries within the current week.
		entryCount, err := countEntriesInWeek(logs, weekStart, weekEnd)
		if err != nil {
			return nil, err
		}

		entryCountPerWeek[weekNumber] = entryCount
		weekNumber++
	}

	return &entryCountPerWeek, nil
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
		for i := startIdx; i < startIdx+7 && i < logs.NRows(); i++ {
			date, err := time.Parse("2006-01-02", logs.Series[dateCol].Value(i).(string))
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

// countValidWeeks counts ands returns the number of valid weeks in a
// given diet phase.
func countValidWeeks(e map[int]int) int {
	count := 0
	for week := 1; week < len(e); week++ {
		if e[week] > 2 {
			count++
		}
	}

	return count
}

// TODO:
// * Case 1 of ending cut and beginning maintenance phase should be its
// own function. Particularily, it should be a helper function that
// encapsulates transistion from a bulk/cut diet to maintenance. That
// is, a helper function to processPhaseTransistion.
// * If they decide to continue, how much do you update threshold?
//
// checkCutThreshold checks if the user has lost too much weight, in
// which the cut is stopped and a maintenance phase begins.
//
// If they has surpassed the maximum threshold for weight loss, the user
// is notified, and offered the change to transistion to another phase.
func checkCutThreshold(u *UserInfo) error {
	// Find the amount of weight the user has lost.
	weightLost := u.Phase.StartWeight - u.Weight
	// Find the theshold weight the user is allowed to lose.
	threshold := u.Phase.StartWeight * 0.10

	// If the user has lost more than 10% of starting weight,
	if weightLost > threshold {
		fmt.Println("You've reached the maximum threshold for weight loss (you've lost more than 10% of your starting weight in a single cutting phase). Stopping your cut and beginning a maintenance phase is highly recommended. Please choose one of the following actions:")

		// Get option.
		var option string
		for {
			fmt.Println("1. End cut and begin maintenance phase")
			fmt.Println("2. Choose a different diet phase.")
			fmt.Println("3. Continue with the cut.")
			fmt.Printf("Enter actions (1, 2, or 3): ")
			fmt.Scanln(&option)

			if option != "1" && option != "2" && option != "3" {
				fmt.Println("Invalid action. Please try again.")
				continue
			}
			break
		}

		switch option {
		case "1":
			// End cut and begin maintenance phase.
			// Note: diet duration is left unmodified so maintenance phase
			// lasts as long as the cut.

			// Set phase to maintenance.
			u.Phase.Name = "maintain"

			// Immediately start maintenance phase.
			u.Phase.StartDate = time.Now()
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
			fmt.Println("Saved user information.")
		case "2": // Change to different phase.
			err := processPhaseTransition(u)
			if err != nil {
				return err
			}
		case "3": // Continue with the cut.
		}
	}

	return nil
}

// checkCutLoss checks to see if user is on the track to meeting weight
// loss goal.
func checkCutLoss(u *UserInfo, logs *dataframe.DataFrame) (WeightLossStatus, float64, error) {
	weeksLostTooMuch := 0   // Consecutive weeks where the user gained too much weight.
	weeksLostTooLittle := 0 // Consecutive weeks where the user gained too little weight.
	totalLossTooMuch := 0.0
	totalLossTooLittle := 0.0

	// Iterate over each week of the diet.
	for date := u.Phase.LastCheckedDate; date.Before(u.Phase.EndDate); date = date.AddDate(0, 0, 7) {
		weekStart := date
		weekEnd := date.AddDate(0, 0, 6)

		totalWeekWeightChange, valid, err := totalWeightChangeWeek(logs, weekStart, weekEnd, u)
		if err != nil {
			return 0, 0, err
		}

		if !valid {
			weeksLostTooLittle = 0
			weeksLostTooMuch = 0
			totalLossTooMuch = 0
			totalLossTooLittle = 0
			continue
		}

		status := metWeeklyGoalCut(u, totalWeekWeightChange)

		switch status {
		case lostTooLittle:
			weeksLostTooLittle++
			totalLossTooLittle += totalWeekWeightChange

			weeksLostTooMuch = 0
			totalLossTooMuch = 0
		case lostTooMuch:
			weeksLostTooMuch++
			totalLossTooMuch += totalWeekWeightChange

			weeksLostTooLittle = 0
			totalLossTooLittle = 0
		case withinLossRange:
			weeksLostTooLittle = 0
			totalLossTooLittle = 0

			weeksLostTooMuch = 0
			totalLossTooMuch = 0
		}

		if weeksLostTooLittle >= 2 {
			return status, totalLossTooLittle, nil
		}

		if weeksLostTooMuch >= 2 {
			return status, totalLossTooMuch, nil
		}
	}

	return withinLossRange, 0, nil
}

// metWeeklyGoalCut checks to see if a given week has met the weekly
// change in weight goal
func metWeeklyGoalCut(u *UserInfo, totalWeekWeightChange float64) WeightLossStatus {
	lowerTolerance := u.Phase.WeeklyChange * 0.2
	upperTolerance := math.Abs(u.Phase.WeeklyChange) * 0.1

	// If user did not lose enough this week,
	if totalWeekWeightChange > u.Phase.WeeklyChange+upperTolerance {
		/*
			fmt.Printf("User did not lose enough this week. total > WeeklyChange+upperTol:   %f < %f\n", totalWeekWeightChange, u.Phase.WeeklyChange-lowerTolerance)
		*/
		return lostTooLittle
	}
	// If user lost too much this week,
	if totalWeekWeightChange < u.Phase.WeeklyChange+lowerTolerance {
		/*
			fmt.Printf("User lost too much this week. total < WeeklyChange+lowerTol:   %f < %f\n", totalWeekWeightChange, u.Phase.WeeklyChange+upperTolerance)
		*/
		return lostTooMuch
	}

	/*
		fmt.Printf("User's change in weight was within range this week. avgWeek (%f) was close enough to WeeklyChange(%f):\n", totalWeekWeightChange, u.Phase.WeeklyChange)
	*/
	return withinLossRange
}

// removeCals calulates the daily caloric deficit and then attempts
// to apply that deficit though first cutting fats, then carbs, and
// finally protein.
//
// The deficit will be applied up to the minimmum macro values.
func removeCals(u *UserInfo, totalWeekWeightChange float64) {

	diff := totalWeekWeightChange - u.Phase.WeeklyChange

	// Get weekly average weight change in calories.
	totalWeekWeightChangeCals := diff * calsPerPound
	// Get daily average weight change in calories.
	avgDayWeightChangeCals := totalWeekWeightChangeCals / 7

	// Set deficit
	deficit := avgDayWeightChangeCals

	// Update calorie goal.
	u.Phase.GoalCalories -= deficit
	fmt.Printf("Reducing caloric deficit by %.2f calories.\n", deficit)
	fmt.Printf("New calorie goal: %.2f.\n", u.Phase.GoalCalories)

	// Convert caloric deficit to fats in grams.
	fatDeficit := deficit * calsInFats

	// If the fat deficit is greater than or equal to the available fats
	// left (up to minimum fats limit), then apply the defict
	// exclusively through removing fats.
	if fatDeficit >= (u.Macros.Fats - u.Macros.MinFats) {
		u.Macros.Fats -= u.Macros.MinFats
		return
	}
	// Otherwise, there are not enough fats to competely apply the deficit, so remove the remaining fats.
	// Remove fats up to the minimum fats limit.
	remainingInFats := u.Macros.MinFats - deficit
	u.Macros.Fats -= deficit - remainingInFats

	// Convert the remaining fats in grams to calories.
	remainingInCals := remainingInFats * calsInFats
	// Convert the remaining calories to carbs in grams.
	carbDeficit := remainingInCals / calsInCarbs

	// If carb deficit is greater than or equal to the availiable carbs
	// left (up to minimum carbs limit), then apply the remaining
	// deficit by removing carbs.
	if carbDeficit >= (u.Macros.Carbs - u.Macros.MinCarbs) {
		u.Macros.Carbs -= u.Macros.MinCarbs
		return
	}
	// Otherwise, remove fats up to the minimum carbs limit.
	remainingInCarbs := u.Macros.MinCarbs - carbDeficit
	u.Macros.Carbs -= carbDeficit - remainingInCarbs

	// Set protein deficit in grams to the carbs that could not be
	// removed.
	// Note: calsInCarbs = calsInProtein.
	proteinDeficit := remainingInCals

	// If protein deficit is greater than or equal to the availiable
	// protein left (up to minimum protein limit), then apply the
	// remaining deficit by removing protein.
	if proteinDeficit >= (u.Macros.Protein - u.Macros.MinProtein) {
		u.Macros.Protein -= u.Macros.MinProtein
		return
	}
	// Otherwise, remove protein up to the minimum protein limit.
	remainingInProtein := u.Macros.MinProtein - proteinDeficit
	u.Macros.Protein -= proteinDeficit - remainingInProtein

	// Convert the remaining protein in grams to calories.
	remaining := remainingInProtein * calsInProtein

	// If the remaining calories are not zero, then stop removing macros
	// and update the diet goal calories.
	if remaining != 0 {
		fmt.Printf("Could not reach a deficit of %.2f as the minimum fat, carb, and protein limits have been met.\n", deficit)
		fmt.Printf("Updating caloric deficit to %.2f\n", deficit-remaining)
		// Override initial cut calorie goal.
		u.Phase.GoalCalories = u.TDEE - deficit + remaining
		fmt.Printf("New calorie goal: %.2f.\n", u.Phase.GoalCalories)
		return
	}
}

// TODO:
// * Case 1 of ending bulk and beginning maintenance phase should be its
// own function. Particularily, it should be a helper function that
// encapsulates transistion from a bulk/cut diet to maintenance. That
// is, a helper function to processPhaseTransistion.
// * If they decide to continue, how much do you update threshold?
//
// checkBulkThreshold checks if the user has gained too much weight, in
// which the bulk is stopped and a maintenance phase begins.
//
// Note: diet duration is left unmodified so maintenance phase
// lasts as long as the bulk and threshold is for beginner's by default.
func checkBulkThreshold(u *UserInfo) error {
	// Find the amount of weight the user has gained.
	weightGain := u.Weight - u.Phase.StartWeight
	// Find the theshold weight the user is allowed to lose.
	threshold := u.Phase.StartWeight * 0.10

	// If the user has lost more than 10% of starting weight,
	if weightGain > threshold {
		fmt.Println("You've reached the maximum threshold for weight gain (you've gained more than 10% of your starting weight in a single bulking phase). Stopping your bulk and beginning a maintenance phase is highly recommended. Please choose one of the following actions:")

		// Get option.
		var option string
		for {
			fmt.Println("1. End bulk and begin maintenance phase")
			fmt.Println("2. Choose a different diet phase.")
			fmt.Println("3. Continue with the bulk.")
			fmt.Printf("Enter actions (1, 2, or 3): ")
			fmt.Scanln(&option)

			if option != "1" && option != "2" && option != "3" {
				fmt.Println("Invalid action. Please try again.")
			}
			break
		}

		switch option {
		case "1":
			// End bulk and begin maintenance phase.
			// Note: diet duration is left unmodified so maintenance phase
			// lasts as long as the bulk.

			// Set phase to maintenance.
			u.Phase.Name = "maintain"

			// Immediately start maintenance phase.
			u.Phase.StartDate = time.Now()
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
			fmt.Println("Saved user information.")
		case "2": // Change to different phase.
			err := processPhaseTransition(u)
			if err != nil {
				return err
			}
		case "3": // Continue with the cut.
		}
	}

	return nil
}

// CheckPhaseStatus checks if the phase is active.
func CheckPhaseStatus(u *UserInfo) (bool, error) {
	t := time.Now()
	// If today comes before diet start date, then phase has not yet begun.
	if t.Before(u.Phase.StartDate) {
		return false, nil
	}

	// If today comes after diet end date, diet phase is over.
	if t.After(u.Phase.EndDate) {
		fmt.Println("Diet phase completed! Starting the diet phase transistion process.")
		// Process phase transition
		err := processPhaseTransition(u)
		if err != nil {
			return false, err
		}

		return u.Phase.Active, nil
	}

	// If today is the first day of the diet (active status has yet to be
	// updated),
	if u.Phase.Active == false {
		// Set starting weight to current weight.
		u.Phase.StartWeight = u.Weight

		// Check if goal weight is still valid.
		_, err := validateGoalWeight(strconv.FormatFloat(u.Phase.GoalWeight, 'f', -1, 64), u)
		if err != nil { // If weight is now invalid,
			option := getNextAction(u)

			switch option {
			case "1": // Get new goal weight.
				u.Phase.GoalWeight = getGoalWeight(u)
				u.Phase.Active = true
			case "2": // Change to different phase.
				err := processPhaseTransition(u)
				if err != nil {
					return false, err
				}
			}
		}
	}
	return u.Phase.Active, nil
}

// getNextAction prompts user for the next action given that they've
// already surpassed their inital weight goal, validates their reponse
// until they've entered a valid next action, and returns the valid action.
func getNextAction(u *UserInfo) (a string) {
	printNextAction(u.Phase.Name)

	for {
		// Prompt for next action.
		a := promptNextAction()

		// Validate user response.
		err := validateNextAction(a)
		if err != nil {
			fmt.Println("Invalid next action. Please try again.")
			continue
		}

		break
	}
	return a
}

// printNextAction prints the options for what to do next given that
// they've already surpassed their inital weight goal.
func printNextAction(phase string) {
	switch phase {
	case "cut":
		fmt.Println("It appears you've already surpassed your inital weight loss goal before starting the diet. Please choose one of the following actions:")
		fmt.Println("1. Set a lower weight loss goal: If you've like to continue the weight loss diet, you can enter a new, lower weight loss goal that is achievable.")
		fmt.Println("2. Choose a different diet phase: If you've already achieved your weight loss goal, you may consider alternativ options such as transitioning to a new diet phase.")
	case "maintain":
		fmt.Println("It appears you've already surpassed your inital weight maintience goal before starting the diet. Please choose one of the following actions:")
		fmt.Println("1. Adjust mantience goal: If you've like to continue the maintience diet, you can enter a new maintenance goal that is achievable.")
		fmt.Println("2. Choose a different diet phase: If you've decided to start a different diet phase, you are free to change to your desired phase.")
	case "bulk":
		fmt.Println("It appears you've already surpassed your inital weight gain goal before starting the diet. Please choose one of the following actions:")
		fmt.Println("1. Set a heigher weight gain goal: If you've like to continue the weight loss diet, you can enter a new, lower weight loss goal that is achievable.")
		fmt.Println("2. Choose a different diet phase: If you've already achieved your weight loss goal, you may consider alternativ options such as transitioning to a new diet phase.")
	}
}

// promptNextAction prompts the user for the next action.
func promptNextAction() (option string) {
	fmt.Printf("Enter actions (1 or 2): ")
	fmt.Scanln(&option)
	return option
}

// validateNextAction validates the next action.
func validateNextAction(a string) error {
	if a != "1" && a != "2" {
		return errors.New("Invalid action.")
	}
	return nil
}

// checkMaintenance ensures user is maintaining the same weight.
func checkMaintenance(u *UserInfo, logs *dataframe.DataFrame) (WeightMaintenanceStatus, float64, error) {
	weeksGained := 0 // Consecutive weeks where the user gained too much weight.
	weeksLost := 0   // Consecutive weeks where the user lost too much weight.
	totalGain := 0.0
	totalLoss := 0.0

	// Iterate over each week of the diet.
	for date := u.Phase.LastCheckedDate; date.Before(u.Phase.EndDate); date = date.AddDate(0, 0, 7) {
		weekStart := date
		weekEnd := date.AddDate(0, 0, 6)

		totalWeekWeightChange, valid, err := totalWeightChangeWeek(logs, weekStart, weekEnd, u)
		if err != nil {
			return 0, 0, err
		}

		if !valid {
			weeksGained = 0
			weeksLost = 0
			totalGain = 0
			totalLoss = 0
			continue
		}

		status := metWeeklyGoalMainenance(u, totalWeekWeightChange)

		switch status {
		case lost:
			weeksLost++
			totalLoss += totalWeekWeightChange

			weeksGained = 0
			totalGain = 0
		case gained:
			weeksGained++
			totalGain += totalWeekWeightChange

			weeksLost = 0
			totalLoss = 0
		case maintained:
			weeksLost = 0
			weeksGained = 0
			totalGain = 0
			totalLoss = 0
		}

		if weeksLost >= 2 {
			return status, totalLoss, nil
		}

		if weeksGained >= 2 {
			return status, totalGain, nil
		}
	}

	return maintained, 0, nil
}

// metWeeklyGoalMainenance checks to see if a given week has met the weekly
// change in weight goal
func metWeeklyGoalMainenance(u *UserInfo, totalWeekWeightChange float64) WeightMaintenanceStatus {
	lowerTolerance := 0.20
	upperTolerance := 0.20

	// If user lost too much weight this week,
	if totalWeekWeightChange < u.Phase.WeeklyChange-lowerTolerance {
		/*
			fmt.Printf("User lost too much this week. total < WeeklyChange-lowerTol:   %f < %f\n", totalWeekWeightChange, u.Phase.WeeklyChange-lowerTolerance)
		*/
		return lost
	}
	// If user gained too much weight this week,
	if totalWeekWeightChange > u.Phase.WeeklyChange+upperTolerance {
		/*
			fmt.Printf("User gained too much this week. total > WeeklyChange+upperTol:   %f > %f\n", totalWeekWeightChange, u.Phase.WeeklyChange+upperTolerance)
		*/
		return gained
	}

	/*
		fmt.Printf("User's change in weight was within range this week. avgWeek (%f) was close enough to WeeklyChange(%f):\n", totalWeekWeightChange, u.Phase.WeeklyChange)
	*/
	return maintained
}

// checkBulkGain checks to see if user is on the track to meeting weight
// gain goal.
func checkBulkGain(u *UserInfo, logs *dataframe.DataFrame) (WeightGainStatus, float64, error) {
	weeksGainedTooMuch := 0   // Consecutive weeks where the user gained too much weight.
	weeksGainedTooLittle := 0 // Consecutive weeks where the user gained too little weight.
	totalGain := 0.0
	totalLoss := 0.0

	for date := u.Phase.LastCheckedDate; date.Before(u.Phase.EndDate); date = date.AddDate(0, 0, 7) {
		weekStart := date
		weekEnd := date.AddDate(0, 0, 6)

		totalWeekWeightChange, valid, err := totalWeightChangeWeek(logs, weekStart, weekEnd, u)
		if err != nil {
			return -2, 0, err
		}

		if !valid {
			weeksGainedTooLittle = 0
			weeksGainedTooMuch = 0
			totalGain = 0
			totalLoss = 0
			continue
		}

		status := metWeeklyGoalBulk(u, totalWeekWeightChange)

		switch status {
		case gainedTooLittle:
			weeksGainedTooLittle++
			totalLoss += totalWeekWeightChange

			weeksGainedTooMuch = 0
			totalGain = 0
		case gainedTooMuch:
			weeksGainedTooMuch++
			totalGain += totalWeekWeightChange

			weeksGainedTooLittle = 0
			totalLoss = 0
		case withinGainRange:
			weeksGainedTooLittle = 0
			weeksGainedTooMuch = 0
			totalGain = 0
			totalLoss = 0
		}

		if weeksGainedTooLittle >= 2 {
			return status, totalLoss, nil
		}

		if weeksGainedTooMuch >= 2 {
			return status, totalGain, nil
		}
	}

	return withinGainRange, 0, nil
}

// metWeeklyGoalBulk checks to see if a given week has met the weekly
// change in weight goal
func metWeeklyGoalBulk(u *UserInfo, totalWeekWeightChange float64) WeightGainStatus {
	lowerTolerance := u.Phase.WeeklyChange * 0.1
	upperTolerance := u.Phase.WeeklyChange * 0.2

	// If user did not gain enough this week,
	if totalWeekWeightChange < u.Phase.WeeklyChange-lowerTolerance {
		/*
			fmt.Printf("User did not gain enough this week. total < WeeklyChange-lowerTol:   %f < %f\n", totalWeekWeightChange, u.Phase.WeeklyChange-lowerTolerance)
		*/
		return gainedTooLittle
	}
	// If user gained too much this week,
	if totalWeekWeightChange > u.Phase.WeeklyChange+upperTolerance {
		/*
			fmt.Printf("User gained too much this week. total < WeeklyChange+upperTol:   %f < %f\n", totalWeekWeightChange, u.Phase.WeeklyChange+upperTolerance)
		*/
		return gainedTooMuch
	}

	/*
		fmt.Printf("User's change in weight was within range this week. avgWeek (%f) was close enough to WeeklyChange(%f):\n", totalWeekWeightChange, u.Phase.WeeklyChange)
	*/
	return withinGainRange
}

// addCals calculates the caloric surplus and then attempts to
// apply it by first adding carbs, then fats, and finally fats.
func addCals(u *UserInfo, totalWeekWeightChange float64) {

	diff := u.Phase.WeeklyChange - totalWeekWeightChange

	// Get weekly average weight change in calories.
	totalWeekWeightChangeCals := diff * calsPerPound
	// Get daily average weight change in calories.
	avgDayWeightChangeCals := totalWeekWeightChangeCals / 7

	// Calculate the needed daily surplus.
	surplus := avgDayWeightChangeCals

	// Update calorie goal.
	u.Phase.GoalCalories += surplus
	fmt.Printf("Adding to caloric surplus by %.2f calories.\n", surplus)
	fmt.Printf("New calorie goal: %.2f.\n", u.Phase.GoalCalories)

	// Convert surplus in calories to carbs in grams.
	carbSurplus := surplus * calsInCarbs

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
	remainingInCals := remainingInCarbs * calsInCarbs
	// Convert the remaining surplus from calories to fats.
	fatSurplus := remainingInCals / calsInFats

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
	remainingInCals = remainingInFats * calsInFats
	// Convert the remaining surplus from calories to protein.
	proteinSurplus := remainingInCals / calsInProtein

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
	remaining := remainingInProtein * calsInProtein

	// If the remaining calories are not zero, then stop adding to macros
	// and update the diet goal calories and return.
	if remaining != 0 {
		fmt.Printf("Could not reach a surplus of %f since the maximum fat, carb, and protein limits were met before the entire surplus could be applied.\n", surplus)
		fmt.Printf("Updating caloric surplus to %f.\n", surplus-remaining)
		// Override initial cut calorie goal.
		u.Phase.GoalCalories = u.TDEE + surplus - remaining
		fmt.Printf("New calorie goal: %.2f.\n", u.Phase.GoalCalories)
		return
	}
}

// totalWeightChangeWeek calculates and returns the total change in
// weight for a given week.
func totalWeightChangeWeek(logs *dataframe.DataFrame, weekStart, weekEnd time.Time, u *UserInfo) (float64, bool, error) {
	var date time.Time
	var err error
	var weight float64
	var startIdx int
	var i int
	totalWeightChangeWeek := 0.0

	// Get the dataframe index of the entry with the start date of the
	// diet.
	startIdx, err = findEntryIdx(logs, weekStart)
	if err != nil {
		return 0, false, err
	}

	// Iterate over each day of the week starting from startIdx.
	for i = startIdx; i < startIdx+7 && i < logs.NRows(); i++ {
		// Get entry date.
		date, err = time.Parse("2006-01-02", logs.Series[dateCol].Value(i).(string))
		if err != nil {
			log.Println("ERROR: Couldn't parse date:", err)
			return 0, false, err
		}

		// If date falls after the end of the week, return setting `valid`
		// week variable to false. This ensures we only consider entry's
		// that are within a full week.
		if date.After(weekEnd) {
			return 0, false, nil
		}

		// Get entry weight.
		w := logs.Series[weightCol].Value(i).(string)
		weight, err = strconv.ParseFloat(w, 64)
		if err != nil {
			log.Println("ERROR: Failed to convert string to float64:", err)
			return 0, false, err
		}

		// Get the previous weight to current day.
		previousWeight, err := getPrecedingWeightToDay(u, logs, weight, i)
		if err != nil {
			return 0, false, err
		}

		// Calculate the weight change between two days.
		weightChange := weight - previousWeight

		// Update total weight change
		totalWeightChangeWeek += weightChange
	}

	// If there were zero entries found in the week, then return early.
	if i == startIdx {
		fmt.Println("Zero entries found this week.")
		return 0, false, nil
	}

	// Update the last checked day in the diet phase to the last day of the
	// week.
	u.Phase.LastCheckedDate = date

	return totalWeightChangeWeek, true, nil
}

// findEntryIdx finds the index of an entry given a date.
func findEntryIdx(logs *dataframe.DataFrame, day time.Time) (int, error) {
	var startIdx int
	// Find the index of weekStart in the data frame.
	for i := 0; i < logs.NRows(); i++ {
		date, err := time.Parse("2006-01-02", logs.Series[dateCol].Value(i).(string))
		if err != nil {
			log.Println("ERROR: Couldn't parse date:", err)
			return 0, err
		}

		if date.Equal(day) {
			startIdx = i
			break
		}

		continue
	}

	return startIdx, nil
}

// getPrecedingWeightToDay returns the preceding entry to a given week.
func getPrecedingWeightToDay(u *UserInfo, logs *dataframe.DataFrame, weight float64, startIdx int) (float64, error) {
	var previousWeight float64
	// If entry is the first in the dataframe, set previous weight
	// equal to zero. This is necessary to prevent index out of bounds
	// error.
	if startIdx == 0 {
		previousWeight = weight
		return previousWeight, nil
	}

	// Get entry date.
	date, err := time.Parse("2006-01-02", logs.Series[dateCol].Value(startIdx-1).(string))
	if err != nil {
		log.Println("ERROR: Couldn't parse date:", err)
		return 0, err
	}

	// If date is before the diet start date,
	if date.Before(u.Phase.StartDate) {
		previousWeight = weight
		return previousWeight, nil
	}
	// Otherwise, the date is after the start date.

	// Get previous entry's weight.
	pw := logs.Series[weightCol].Value(startIdx - 1).(string)
	previousWeight, err = strconv.ParseFloat(pw, 64)
	if err != nil {
		log.Println("ERROR: Failed to convert string to float64:", err)
		return 0, err
	}

	return previousWeight, nil
}

// TODO: Handle case where user is brand new. They set diet date start
// in the future. They don't log any information.
//
// processPhaseTransition prompts user for the diet phase to transistion
// to, validated their response until they enter a valid transistion
// option, savesd next phase to config file, and returns error to indicate success or failure.
func processPhaseTransition(u *UserInfo) error {
	fmt.Println("Step 1: Diet phase recap")
	fmt.Printf("Goal weight: %f. Current weight: %f\n", u.Phase.GoalWeight, u.Weight)

	printTransitionSuggestion(u.Phase.Name)

	processUserInfo(u)

	// TODO: If the user does follow recommendation of a maintenance
	// phase coming from a bulk that has just ended, then
	// 1. Calculate your weekly calorie surplus based on the average weight
	// gain over the final two weeks of bulking.
	// 2. Start out phase by decreasing your caloric intake by that amount.

	// Save user info to config file.
	err := saveUserInfo(u)
	if err != nil {
		log.Println("Failed to save user info:", err)
		return err
	}
	log.Println("User info saved successfully.")

	return nil
}

// printTransitionSuggestion prints the suggested diet phase to
// transistion into given the diet phase that is ending.
func printTransitionSuggestion(phase string) {
	switch phase {
	case "cut":
		fmt.Println("After a fully completed cut phase, a maintenance phase of the same duration as your completed cut is recommended.")
	case "maintain":
		fmt.Println("After a fully completed maintenance phase, you are primed for a bulk or a cut. There's also nothing inherently wrong with extending the maintenance phase, you may just be losing out on time that could be used for building muscle or losing fat.")
	case "bulk":
		fmt.Println("After a fully completed bulk phase, a maintenance phase of the at least a month is recommended.")
	}
}

// processUserInfo executes the common operations for handling user
// information. It sets the diet phase, determines minimum and maximum
// diet duration, calculates macros, prompts for confirmation, and
// updates the user information.
func processUserInfo(u *UserInfo) {
	// Get the phase the user wants to start.
	u.Phase.Name = getDietPhase()

	// Set initial start weight.
	// Note: If diet is sometime in the future, this field will be upated
	// to the weight of the user when the user begins the diet.
	u.Phase.StartWeight = u.Weight

	getPhaseInfo(u)

	// Set min and max diet phase duration.
	setMinMaxPhaseDuration(u)

	// Set min and max values for macros.
	setMinMaxMacros(u)

	// Set suggested macro split.
	protein, carbs, fats := calculateMacros(u)
	u.Macros.Protein = protein
	u.Macros.Carbs = carbs
	u.Macros.Fats = fats

	// Print new phase information to user.
	promptConfirmation(u)
}

// getPhaseInfo prompts user for information to initialize the Phase
// struct, validates their response until they enter valid values, and
// sets their value to the corresponding struct field. Some fields are
// simply calculated.
func getPhaseInfo(u *UserInfo) {
	// Set min and max diet durations.
	setMinMaxPhaseDuration(u)

	// Fill out remaining userInfo struct fields given user preference on
	// recommended or custom diet pace.
	switch getDietChoice(u) {
	case "recommended":
		handleRecommendedDiet(u)
	case "custom":
		handleCustomDiet(u)
	}
}

// getDietChoice prompts user for their diet choice, validates their
// reponse until they enter a valid diet choice, and returns the valid
// diet choice.
func getDietChoice(u *UserInfo) string {
	fmt.Println("Step 3: Choose diet goal.")

	// Print to user recommended and custom diet goal options.
	printDietChoices(u.Phase.Name)

	var c string
	for {
		// Prompt user for diet goal.
		c = promptDietChoice()

		// Validate user response.
		err := validateDietChoice(c)
		if err != nil {
			fmt.Println("Invalid diet goal. Please try again.")
			continue
		}

		break
	}

	return c
}

// printDietChoices prints recommended and custom diet options.
func printDietChoices(phase string) {
	fmt.Printf("Recommended: ")
	switch phase {
	case "cut":
		fmt.Printf("Lose 4 lbs in 8 weeks.\n")
	case "maintain":
		fmt.Printf("Maintain same weight for 5 weeks.\n")
	case "bulk":
		fmt.Printf("Gain 2.5 lbs in 10 weeks.\n")
	}

	fmt.Println("Custom: Choose diet duration and rate of weight change.")
}

// promptDietChoice prints diet goal options, prompts for diet goal,
// and validates user response.
func promptDietChoice() (c string) {
	fmt.Printf("Enter diet choice (recommended or custom): ")
	fmt.Scanln(&c)
	return c
}

// validateDietChoice validates and returns user diet choice.
func validateDietChoice(c string) error {
	if c != "recommended" && c != "custom" {
		return errors.New("Invalid diet choice.")
	}
	return nil
}

// handleRecommendedDiet sets UserInfo struct fields according to a
// reccomended diet.
func handleRecommendedDiet(u *UserInfo) {
	// Get the diet start date.
	u.Phase.StartDate = getStartDate(u)

	// TODO: show logic that derives daily calories.
	// Calculate daily caloric change need to create a deficit/surplus.
	c := u.Phase.WeeklyChange * 500

	// Calculate expected change in weight for cut/bulk.
	a := u.Phase.StartWeight * u.Phase.WeeklyChange

	switch u.Phase.Name {
	case "cut":
		// Set WeeklyChange, Duration, GoalWeight, and GoalCalories.
		setRecommendedValues(u, -1, 8, u.Phase.StartWeight-a, u.TDEE-c)
	case "maintain":
		// Set WeeklyChange, Duration, GoalWeight, and GoalCalories.
		setRecommendedValues(u, 0, 5, u.Phase.StartWeight, u.TDEE)
	case "bulk":
		// Set WeeklyChange, Duration, GoalWeight, and GoalCalories.
		setRecommendedValues(u, 0.25, 10, u.Phase.StartWeight+a, u.TDEE+c)
	}

	// Calculate the diet end date.
	u.Phase.EndDate = calculateEndDate(u.Phase.StartDate, u.Phase.Duration)
}

// setRecommendedValues sets the recommended values for the UserInfo
// fields: weekly weight change, diet duration, diet goal weight, and
// diet daily calories.
func setRecommendedValues(u *UserInfo, w, d, g, c float64) {
	u.Phase.WeeklyChange = w
	u.Phase.Duration = d
	u.Phase.GoalWeight = g
	u.Phase.GoalCalories = c
	// Initialize last checked date.
	u.Phase.LastCheckedDate = u.Phase.StartDate
}

// calculateEndDate calculates the diet end date given diet start date
// and diet duration in weeks.
func calculateEndDate(d time.Time, duration float64) time.Time {
	endDate := d.AddDate(0, 0, int(duration*7.0))
	return endDate
}

// handleCustomDiet sets UserInfo struct fields according to custom diet
// specified by the user.
func handleCustomDiet(u *UserInfo) {
	// Get diet start date.
	u.Phase.StartDate = getStartDate(u)

	// Initialize last checked date.
	u.Phase.LastCheckedDate = u.Phase.StartDate

	// set diet end date.
	setEndDate(u)

	// Get diet goal weight.
	u.Phase.GoalWeight = getGoalWeight(u)

	// Calculate weekly weight change rate.
	u.Phase.WeeklyChange = calculateWeeklyChange(u.Weight, u.Phase.GoalWeight, u.Phase.Duration)

	// Calculate daily caloric change needed for cut or bulk.
	change := u.Phase.WeeklyChange * 500

	switch u.Phase.Name {
	case "cut":
		u.Phase.GoalCalories = u.TDEE - change
	case "maintain":
		u.Phase.GoalCalories = u.TDEE
	case "bulk":
		u.Phase.GoalCalories = u.TDEE + change
	}
}

// getStartDate prompts user for diet start date, validates user response
// until user enters valid date, and returns valid date.
func getStartDate(u *UserInfo) (date time.Time) {
	for {
		// Prompt user for diet start date.
		r := promptDate("Enter diet start date (YYYY-MM-DD) [Press Enter for today's date]: ")

		u.Phase.Active = false // set active to false by default.
		// If user entered default date,
		if r == "" {
			// set date to today's date.
			r = time.Now().Format("2006-01-02")
			// Set phase status to true.
			u.Phase.Active = true
		}

		// Validate user response.
		var err error
		date, err = validateDate(r)
		if err != nil {
			fmt.Printf("%v. Please try again.\n", err)
			continue
		}

		break
	}
	return date
}

// setEndDate prompts user for diet end date, validates user response
// until user enters valid date, and returns valid date.
func setEndDate(u *UserInfo) {
	for {
		// Prompt user for diet end date.
		r := promptDate("Enter diet end date (YYYY-MM-DD): ")

		// Validate user response.
		date, duration, err := validateEndDate(r, u)
		if err != nil {
			fmt.Printf("%v. Please try again.\n", err)
			continue
		}

		u.Phase.EndDate = date
		u.Phase.Duration = duration

		break
	}
}

// promptDate prompts and returns diet start date.
func promptDate(promptStr string) string {
	reader := bufio.NewReader(os.Stdin)
	// Prompt user for diet stop date.
	fmt.Printf("%s\n", promptStr)
	response, _ := reader.ReadString('\n')

	return strings.TrimSpace(response)
}

// validateEndDate prompts user for diet end date, validates user
// response, and returns diet end date and diet duration when date is
// valid.
func validateEndDate(r string, u *UserInfo) (time.Time, float64, error) {
	// Ensure user response is a date.
	d, err := validateDate(r)
	if err != nil {
		return time.Time{}, 0, errors.New("Invalid date.")
	}

	// Calculate diet duration in weeks given start and end date.
	dur := calculateDuration(u.Phase.StartDate, d).Hours() / 24 / 7

	// Does end date fall after start date?
	if d.Before(u.Phase.StartDate) {
		return time.Time{}, 0, errors.New("Invalid diet phase end date. End date must be after diet start date.")
	}

	// Is diet duration less than max diet duration?
	if dur > u.Phase.MaxDuration {
		e := fmt.Sprintf("Invalid diet phase end date. Diet duration of %.2f weeks exceeds the maximum duration of %.2f.", math.Round(dur*100)/100, u.Phase.MaxDuration)
		return time.Time{}, 0, errors.New(e)
	}

	// Is diet duration greater than min diet duration?
	if dur < u.Phase.MinDuration {
		e := fmt.Sprintf("Invalid diet phase end date. Diet duration of %.2f weeks falls short of the minimum duration of %.2f.", math.Round(dur*100)/100, u.Phase.MinDuration)
		return time.Time{}, 0, errors.New(e)
	}

	return d, dur, nil
}

// validateDate validates the given date string and returns date if
// valid.
func validateDate(dateStr string) (time.Time, error) {
	// Validate user response.
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}, err
	}

	return date, nil
}

// calculateDuration calculates and returns diet duration as a
// `time.Duration` value given start and end date.
func calculateDuration(start, end time.Time) time.Duration {
	d := end.Sub(start)
	return d
}

// getGoalWeight prompts user for goal weight, validates their response
// until they enter a valid goal weight, and returns valid goal weight.
func getGoalWeight(u *UserInfo) (g float64) {
	// If phase is maintenance, return starting weight and skip prompting.
	if u.Phase.Name == "maintain" {
		return u.Phase.StartWeight
	}

	for {
		// Prompt user for goal weight.
		w := promptGoalWeight()

		// Validate user response.
		var err error
		g, err = validateGoalWeight(w, u)
		if err != nil {
			fmt.Println("Invalid goal weight. Please try again.")
			continue
		}

		break
	}

	return g
}

// promptGoalWeight prompts and returns user goal weight.
func promptGoalWeight() (w string) {
	fmt.Printf("Enter your goal weight: ")
	fmt.Scanln(&w)
	return w
}

// validateGoalWeight prompts validates diet goal weight.
// Maintenance phase goal weight need not be validated as it is just
// set to the users starting weight.
func validateGoalWeight(weightStr string, u *UserInfo) (g float64, err error) {
	// Convert string to float64.
	g, err = strconv.ParseFloat(weightStr, 64)
	if err != nil || g < 0 {
		return 0, errors.New("Invalid goal weight. Goal weight must be a number.")
	}

	if g == u.Phase.StartWeight {
		return 0, errors.New("Invliad goal weight. For any diet phase other than maintenance, goal weight must differ from starting weight.")
	}

	switch u.Phase.Name {
	case "cut":
		if g > u.Phase.StartWeight {
			return 0, errors.New("Invalid goal weight. For a cut, goal weight must be lower than starting weight.")
		}

		lowerBound := u.Phase.StartWeight * 0.10
		if g < u.Phase.StartWeight-lowerBound {
			return 0, errors.New("Invalid goal weight. For a cut, goal weight cannot be less than 10% of starting body weight.")
		}
	case "bulk":
		if g < u.Phase.StartWeight {
			return 0, errors.New("Invalid goal weight. For a bulk, goal weight must be greater than starting weight.")
		}

		upperBound := u.Phase.StartWeight * 0.10
		if g > u.Phase.StartWeight+upperBound {
			return 0, errors.New("Invalid goal weight. For a bulk, goal weight cannot exceed 10% of starting body weight.")
		}
	}

	return g, nil
}

// calculateWeeklyChange calculates and returns the weekly weight
// change in pounds given current weight, goal weight, and diet duration.
func calculateWeeklyChange(current, goal, duration float64) float64 {
	weeklyChange := (goal - current) / duration
	return weeklyChange
}

// setMinMaxPhaseDuration sets the minimum and maximum diet phase
// duration given the current phase the user has chosen.
func setMinMaxPhaseDuration(u *UserInfo) {
	switch u.Phase.Name {
	case "cut":
		u.Phase.MaxDuration = 12
		u.Phase.MinDuration = 6
	case "maintain":
		u.Phase.MaxDuration = math.Inf(1)
		u.Phase.MinDuration = 0
	case "bulk":
		u.Phase.MaxDuration = 16
		u.Phase.MinDuration = 6
	}
}

// promptConfirmation prints diet summary to the user.
func promptConfirmation(u *UserInfo) {
	// Display current information to the user.
	fmt.Println("Summary:")
	fmt.Println("Diet start date:", u.Phase.StartDate.Format("2006-01-02"))
	fmt.Println("Diet end date:", u.Phase.EndDate.Format("2006-01-02"))
	fmt.Println("Diet duration:", u.Phase.Duration)

	switch u.Phase.Name {
	case "cut":
		fmt.Printf("Target weight: %.2f (%.2f lbs)\n", u.Phase.GoalWeight, u.Phase.StartWeight-u.Phase.GoalWeight)
		fmt.Println("During your cut, you should lean slightly on the side of doing more high-volume training.")
	case "maintain":
		fmt.Printf("Target weight: %.2f\n", u.Phase.GoalWeight)
		fmt.Println("During your maintenance, you should lean towards low-volume training (3-10 rep strength training). Get active rest (barely any training and just living life for two weeks is also an option). This phase is meant to give your body a break to recharge for future hard  training.")
	case "bulk":
		fmt.Printf("Target weight: %.2f (+%.2f lbs)\n", u.Phase.GoalWeight, u.Phase.GoalWeight-u.Phase.StartWeight)
		fmt.Println("During your bulk, you can just train as you normally would.")
	}
}

// getDietPhase prints diet phases, prompts user for diet phase, validates
// user response, and returns valid diet phase.
func getDietPhase() (p string) {
	fmt.Println("Step 2: Choose diet phase.")
	printDietPhases()

	for {
		// Prompt user for diet phase.
		p = promptDietPhase()

		// Validate diet phase.
		err := validateDietPhase(p)
		if err != nil {
			fmt.Println("Invalid diet phase. Please try again.")
			continue
		}

		break
	}

	return p
}

// printDietPhases prints available diet phases and their descriptions.
func printDietPhases() {
	fmt.Println("Fat loss (cut). Lose fat while losing weight and preserving muscle.")
	fmt.Println("Maintenance (maintain). Stay at your current weight.")
	fmt.Println("Muscle gain (bulk). Gain muscle while minimizing fat.")
}

// promptUserPhase prompts the user to enter desired diet phase.
func promptDietPhase() (s string) {
	fmt.Print("Enter phase (cut, maintain, or bulk): ")
	fmt.Scanln(&s)
	return s
}

// validateDietPhase validates user diet phase.
func validateDietPhase(s string) error {
	// If user response is either "cut", "maintain", or "bulk",
	if s == "cut" || s == "maintain" || s == "bulk" {
		return nil
	}

	return errors.New("Invalid diet phase.")
}

// TODO
func Summary() {
	fmt.Println("Generating summary.")
	// Print day summary
	// Print week summary
	// Print month summary
}
