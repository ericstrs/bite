package calories

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	weightCol       = 0               // Dataframe column for weight.
	calsCol         = 1               // Dataframe column for calories.
	dateCol         = 2               // Dataframe column for weight.
	calsInProtein   = 4               // Calories per gram of protein.
	calsInCarbs     = 4               // Calories per gram of carbohydrate.
	calsInFats      = 9               // Calories per gram of fat.
	ConfigFilePath  = "./config.yaml" // Path to user config file.
	EntriesFilePath = "./data.csv"    // Path to user entries file.
)

type UserInfo struct {
	Sex           string    `yaml:"sex"`
	Weight        float64   `yaml:"weight"` // lbs
	Height        float64   `yaml:"height"` // cm
	Age           int       `yaml:"age"`
	ActivityLevel string    `yaml:"activity_level"`
	TDEE          float64   `yaml:"tdee"`
	Macros        Macros    `taml:"macros"`
	System        string    `yaml:"system"`
	Phase         PhaseInfo `yaml:"phase"`
}

type Macros struct {
	Protein    float64 `yaml:"protein"`
	MinProtein float64 `yaml:"min_protein"`
	MaxProtein float64 `yaml:"max_protein"`
	Carbs      float64 `yaml:"carbs"`
	MinCarbs   float64 `yaml:"min_carbs"`
	MaxCarbs   float64 `yaml:"max_carbs"`
	Fats       float64 `yaml:"fats"`
	MinFats    float64 `yaml:"min_fats"`
	MaxFats    float64 `yaml:"max_fats"`
}

// Save user information to yaml config file.
func saveUserInfo(u *UserInfo) error {
	data, err := yaml.Marshal(u)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(ConfigFilePath, data, 0644)
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

// lbsToKg converts pounds to kilograms.
func lbsToKg(lbs float64) float64 {
	return lbs * 0.45359237
}

// kgToLbs converts kilograms to pounds.
func kgToLbs(kg float64) float64 {
	return kg / 0.45359237
}

// inchesToCm converts inches to centimeters.
func inchesToCm(inches float64) float64 {
	return inches * 2.54
}

// cmToInches converts centimeters to inches.
func cmToInches(cm float64) float64 {
	return cm / 2.54
}

// Converts height from cm to feet and inches.
func cmToFeetInches(cm float64) (int, float64) {
	totalInches := cm / 2.54
	feet := int(totalInches / 12)
	inches := math.Mod(totalInches, 12)
	return feet, inches
}

// Converts height from feet and inches to cm.
func feetInchesToCm(feet int, inches float64) float64 {
	totalInches := (float64(feet) * 12) + inches
	return totalInches * 2.54
}

// Converts height from inches to feet and inches.
func inchesToFeetInches(inches float64) (int, float64) {
	feet := int(inches / 12)
	inchesRemainder := math.Mod(inches, 12)
	return feet, inchesRemainder
}

// Converts height from feet and inches to inches.
func feetInchesToInches(feet int, inches float64) float64 {
	return (float64(feet) * 12) + inches
}

// Mifflin calculates and returns the Basal Metabolic Rate (BMR) which is
// based on weight (kg), height (cm), age (years), and sex.
func Mifflin(u *UserInfo) float64 {
	// Convert weight in pounds to kilograms.
	weight := lbsToKg(u.Weight)
	// Convert height in inches to centimeters.
	height := inchesToCm(u.Height)

	factor := 5
	if u.Sex == "female" {
		factor = -151
	}

	var bmr float64
	bmr = (10 * weight) + (6.25 * height) - (5 * float64(u.Age)) + float64(factor)
	return bmr
}

// TDEE calcuates the Total Daily Energy Expenditure (TDEE) based on the
// BMR and user's activity level.
func TDEE(bmr float64, a string) float64 {
	al, err := activity(a)
	if err != nil {
		fmt.Println("ERROR:", err)
		return -1
	}

	return bmr * al
}

// calculateMacros calculates and returns the recommended macronutrients given
// user weight (lbs) and daily caloric intake.
//
// Macronutrients are prioritized in the following way:
// protein > carbs > fats. This may result in an unbalanced
// macro split where fats and carbs are at their minimum, but protein
// sits at its optimal value.
func calculateMacros(u *UserInfo) (float64, float64, float64) {
	minProtein, minCarbs, minFats := handleMinMacros(u)
	if minProtein != 0 && minCarbs != 0 && minFats != 0 {
		return minProtein, minCarbs, minFats
	}

	maxProtein, maxCarbs, maxFats := handleMaxMacros(u)
	if maxProtein != 0 && maxCarbs != 0 && maxFats != 0 {
		return maxProtein, maxCarbs, maxFats
	}

	// Calculate optimal protein and carb amounts.
	protein := 1 * u.Weight
	carbs := 1.5 * u.Weight

	totalCals := (protein * calsInProtein) + (carbs * calsInCarbs)

	remainingCals := u.Phase.GoalCalories - totalCals

	fats := remainingCals / 9

	/*
		fmt.Printf("%f < protein=%f < %f\n%f < carbs=%f < %f\n%f < fats=%f < %f\n",
			u.Macros.MinProtein, protein, u.Macros.MaxProtein, u.Macros.MinCarbs, carbs, u.Macros.MaxCarbs, u.Macros.MinFats, fats, u.Macros.MaxFats)
	*/

	// If fat calculation is less than minimum allowed fats,
	if u.Macros.MinFats > fats {
		fmt.Println("Fats are below minimum limit. Taking calories from carbs and moving them to fats.")
		// move calories from carbs to fats in an attempt to reach minimum
		// fat.

		fatsNeeded := u.Macros.MinFats - fats

		/*
			fmt.Printf("fatsNeeded := u.MinFats - fats. %f := %f - %f\n", fatsNeeded, u.Macros.MinFats, fats)
		*/

		fatCalsNeeded := fatsNeeded * 9
		carbsToRemove := fatCalsNeeded / 4

		// If we are able to remove carbs and still stay above the minimum
		// carb limit,
		if carbs-carbsToRemove > u.Macros.MinCarbs {
			// then we are safe to take the needed calories from carbs and put
			// them towards reaching the minimum fat limit.
			carbs -= carbsToRemove
			fats += fatsNeeded
			return protein, carbs, fats
		}
		// Otherwise, we remove what carbs we can and then take attempt to
		// take the remaining calories from protein in an effor to maintain
		// the minimum fat limit.

		fmt.Println("Minimum carb limit reached and fats are still under minimum amount. Attempting to take calories from protein and move them to fats.")

		// Calculate the carbs we can take away before reaching minimum carb
		// limit.
		carbsToRemove = carbs - u.Macros.MinCarbs

		// Remove the carbs.
		carbs -= carbsToRemove

		// Calculate the calories in carbs that were removed.
		carbsRemovedInCals := carbsToRemove * calsInCarbs

		// Update fats using the carbs that were able to be taken.
		fats += carbsRemovedInCals / calsInFats

		// Calculate the remaining fats need to reach minimum limit.
		fatsNeeded = u.Macros.MinFats - fats

		/*
			fmt.Printf("fatsNeeded := u.MinFats - fats. %f := %f - %f\n", fatsNeeded, u.Macros.MinFats, fats)
		*/

		fatCalsNeeded = fatsNeeded * calsInFats

		// Convert the remaining calories into protein.
		proteinToRemove := fatCalsNeeded / calsInProtein

		// Attempt to take the remaining calories from protein.
		if protein-proteinToRemove > u.Macros.MinProtein {
			// then we are safe to take the needed calories from the protein
			// and put them towards reaching the minimum fat limit.
			protein -= proteinToRemove
			fats += fatsNeeded
			return protein, carbs, fats
		}
		// Otherwise, we have reached the minimum carb and protein limit.

		// Calculate the protein we are allowed to remove.
		proteinToRemove = protein - u.Macros.MinProtein

		// Remove the protein.
		protein -= proteinToRemove

		// Calculate the calories in protein that were removed.
		proteinRemovedInCals := proteinToRemove * calsInProtein

		// Update fats using the protein that were able to be taken.
		fats += proteinRemovedInCals / 9

		/*
			// Calculate the remaining fats needed to reach the minimum limit.
			fatsNeeded = u.Macros.MinFats - fats
			fmt.Printf("fatsNeeded := u.MinFats - fats. %f := %f - %f\n", fatsNeeded, u.Macros.MinFats, fats)
		*/
	}

	// If fat caculation is greater than maximum allowed fats.
	if u.Macros.MaxFats < fats {
		// move calories from carbs and add to fats to reach maximum fats.

		fmt.Println("Calculated fats are above maximum amount. Taking calories from fats and moving them to carbs.")

		fatsToRemove := fats - u.Macros.MaxFats

		/*
			fmt.Printf("fatsToRemove := fats - u.Macros.MaxFats. %f := %f - %f\n", fatsToRemove, fats, u.Macros.MaxFats)
		*/

		fatsToRemoveCals := fatsToRemove * calsInFats
		carbsToAdd := fatsToRemoveCals / calsInCarbs

		// If we are able to adds carbs and still stay below the maximum
		// carb limit,
		if carbs+carbsToAdd < u.Macros.MaxCarbs {
			// then we are safe to adds to carbs.
			fats -= fatsToRemove
			carbs += carbsToAdd
			return protein, carbs, fats
		}
		// Otherwise, we remove what carbs we can and then take attempt to
		// take the remaining calories from protein in an effor to maintain
		// the minimum fat limit.

		fmt.Println("Carb maximum limit reached and fats are still over maximum amount. Attempting to take calories from fats and move them to protein.")

		// Calculate the carbs we can add before reaching maximum carb limit.
		carbsToAdd = u.Macros.MaxCarbs - carbs

		// Add the carbs.
		carbs += carbsToAdd

		// Calculate the calories in carbs that were added.
		carbsAddedInCals := carbsToAdd * calsInCarbs

		// Update fats using the carbs that were able to be added.
		fats -= carbsAddedInCals / calsInFats

		// Calculate the remaining fats need to be added reach maximum limit.
		fatsToRemove = fats - u.Macros.MaxFats

		/*
			fmt.Printf("fatsToRemove := fats - u.Macros.MaxFats. %f := %f - %f\n", fatsToRemove, fats, u.Macros.MaxFats)
		*/

		fatsToRemoveCals = fatsToRemove * calsInFats

		// Convert the remaining calories into protein.
		proteinToAdd := fatsToRemoveCals / calsInProtein

		// Attempt to adds the excess fats calories to protein.
		if protein+proteinToAdd < u.Macros.MaxProtein {
			// then we are safe to adds the excess fat to protein.
			fats -= fatsToRemove
			protein += proteinToAdd
			return protein, carbs, fats
		}
		// Otherwise, we have reached the each maximum carb and protein
		// limit.

		// Calculate the protein we are allowed to add.
		proteinToAdd = u.Macros.MaxProtein - protein

		// Add the protein.
		protein += proteinToAdd

		// Calculate the calories in protein that were added.
		proteinAddedInCals := proteinToAdd * calsInProtein

		// Update fats using the protein that were able to be added.
		fats += proteinAddedInCals / calsInFats

		/*
			// Calculate the remaining fats needed to reach the maximum limit.
			fatsToRemove = fats - u.Macros.MaxFats
			fmt.Printf("fatsToRemove := fats - u.Macros.MaxFats. %f := %f - %f\n", fatsToRemove, fats, u.Macros.MaxFats)
		*/
	}

	// Round macros to two decimal places
	protein = math.Round(protein*100) / 100
	carbs = math.Round(carbs*100) / 100
	fats = math.Round(fats*100) / 100

	return protein, carbs, fats
}

// handleMinMacros checks to see if the minimum macronutrient values in
// calories exceeds the original calorie goal.
func handleMinMacros(u *UserInfo) (float64, float64, float64) {
	// Get the calories from the minimum macro values.
	mc := getMacroCals(u.Macros.MinProtein, u.Macros.MinCarbs, u.Macros.MinFats)
	// If minimum macro values in calories is greater than the daily
	// calorie goal,
	if mc > u.Phase.GoalCalories {
		// Let the user know their daily calories are too low and update
		// daily calories to the minimum allowed.
		fmt.Printf("Minimum macro values in calories exceed original calorie goal of %.2f\n", u.Phase.GoalCalories)

		// Update phase daily goal calories.
		u.Phase.GoalCalories = mc
		fmt.Println("New daily calorie goal:", u.Phase.GoalCalories)

		return u.Macros.MinProtein, u.Macros.MinCarbs, u.Macros.MinFats
	}

	// Indicates calorie goal is greater than the calories in the minimum
	// macros.
	return 0, 0, 0
}

// handleMaxMacros checks to see if the maximum macronutrient values in
// calories exceeds the  original calorie goal.
func handleMaxMacros(u *UserInfo) (float64, float64, float64) {
	// Get the calories from the maximum macro values.
	mc := getMacroCals(u.Macros.MaxProtein, u.Macros.MaxCarbs, u.Macros.MaxFats)
	if mc < u.Phase.GoalCalories {
		// Let the user know their daily calories are too high and update
		// daily calories to the maximum allowed.
		fmt.Printf("Maximum macro values exceed original calorie goal of %f\n", u.Phase.GoalCalories)

		// Update phase daily goal calories.
		u.Phase.GoalCalories = mc
		fmt.Println("New daily calorie goal:", u.Phase.GoalCalories)

		return u.Macros.MaxProtein, u.Macros.MaxCarbs, u.Macros.MaxFats
	}

	// Indicates calorie goal is greater than the calories in the maximum
	// macros.
	return 0, 0, 0
}

// getMacroCals calculates and returns the amount of calories given macronutrients.
func getMacroCals(protein, carbs, fats float64) float64 {
	return (protein * calsInProtein) + (carbs * calsInCarbs) + (fats * calsInFats)
}

// setMinMaxMacros calculates the minimum and maximum macronutrient in
// grams using user's most recent logged bodyweight (lbs).
//
// Note: Min and max values are determined for general health. More
// active lifestyles will likely require different values.
func setMinMaxMacros(u *UserInfo) {
	// Minimum protein daily intake needed for health is about
	// 0.3g of protein per pound of bodyweight.
	u.Macros.MinProtein = 0.3 * u.Weight
	// Maximum protein intake for general health is 2g of protein per
	// pound of bodyweight.
	u.Macros.MaxProtein = 2 * u.Weight

	// Minimum carb daily intake needed for *health* is about
	// 0.3g of carb per pound of bodyweight.
	u.Macros.MinCarbs = 0.3 * u.Weight
	// Maximum carb intake for general health is 4g of carb per pound of
	// bodyweight.
	u.Macros.MaxCarbs = 4 * u.Weight

	// Minimum daily fat intake is 0.3g per pound of bodyweight.
	u.Macros.MinFats = 0.3 * u.Weight
	// Maximum daily fat intake is keeping calorie contribtions from fat
	// to be under 40% of total daily calories .
	u.Macros.MaxFats = 0.4 * u.Phase.GoalCalories / calsInFats
}

// PrintMetrics prints user TDEE, suggested macro split, and generates
// plots using logs data frame.
func PrintMetrics(u *UserInfo) {
	// Get BMR.
	bmr := Mifflin(u)
	fmt.Printf("BMR: %.2f\n", bmr)

	// Get TDEE.
	t := TDEE(bmr, u.ActivityLevel)
	fmt.Printf("TDEE: %.2f\n", t)

	// Get suggested macro split.
	protein, carbs, fats := calculateMacros(u)
	fmt.Printf("Protein: %.2fg Carbs: %.2fg Fats: %.2fg\n", protein, carbs, fats)

	// Create plots
}

// getUserInfo prompts for user details.
func getUserInfo(u *UserInfo) {
	fmt.Println("Step 1: Your details.")

	u.System = "imperial"
	s := getSystem()
	if s == "1" {
		u.System = "metric"
	}

	u.Sex = getSex()

	// Error need not be checked since `u.System` will always either be
	// "imperial" or "metric".
	u.Weight, _ = getWeight(u.System)
	u.Height, _ = getHeight(u.System)

	u.Age = getAge()
	u.ActivityLevel = getActivity()

	// Get BMR
	bmr := Mifflin(u)

	// Set TDEE
	u.TDEE = TDEE(bmr, u.ActivityLevel)
}

// getSystem prompts user for their preferred measurement system,
// validates their response, and returns vaild measurement system.
func getSystem() (s string) {
	for {
		// Prompt user for their preferred measurement system.
		s = promptSystem()

		err := validateSystem(s)
		if err != nil {
			fmt.Println("Invalid option. Please try again.")
			continue
		}

		break
	}
	return s
}

// promptSystem prompts and returns user's preferred measurement system.
func promptSystem() (s string) {
	fmt.Println("Set measurement system to:")
	fmt.Println("1. Metric (kg/cm)")
	fmt.Println("2. Imperial (lbs/inches)")
	fmt.Printf("Type number and <Enter>: ")
	fmt.Scanln(&s)
	return s
}

// validateSystem and returns the user's preferred measurement system.
func validateSystem(s string) error {
	s = strings.ToLower(s)
	if s == "1" || s == "2" {
		return nil
	}

	return errors.New("Invalid option.")
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
	fmt.Print("Enter sex (male/female): ")
	fmt.Scanln(&s)
	return s
}

// validateSex validates user sex and returns sex if valid.
func validateSex(s string) error {
	s = strings.ToLower(s)
	if s == "male" || s == "female" {
		return nil
	}

	return errors.New("Invalid sex.")
}

// getWeight prompts user for weight, validate their response, and
// returns valid weight.
func getWeight(system string) (float64, error) {
	var weight float64
	var err error
	for {
		switch system {
		case "metric":
			fmt.Print("Enter weight (kgs): ")
			_, err = fmt.Scan(&weight)
			if err != nil {
				fmt.Printf("Error reading weight: %v. Please try again.\n", err)
				continue
			}

			weight = kgToLbs(weight)
		case "imperial":
			fmt.Print("Enter weight (lbs): ")
			_, err = fmt.Scan(&weight)
			if err != nil {
				fmt.Printf("Error reading weight: %v. Please try again.\n", err)
				continue
			}
		default:
			return 0, fmt.Errorf("Invalid measurement system: %s", system)
		}

		break
	}

	return weight, nil
}

// getHeight prompts user for height, validates their response, and
// returns their height in inches.
func getHeight(system string) (float64, error) {
	var height float64
	var err error
	for {
		switch system {
		case "metric":
			fmt.Print("Enter height (cm): ")
			_, err = fmt.Scan(&height)

			if err != nil {
				fmt.Printf("Error reading height: %v. Please try again.\n", err)
				continue
			}

			height = cmToInches(height)
		case "imperial":
			// Prompt for feet portion.
			fmt.Print("What is your height (feet portion)? ")
			var feet int
			_, err := fmt.Scan(&feet)
			if err != nil {
				fmt.Printf("Error reading feet: %v. Please try again.", err)
				continue
			}

			// Prompt for inches portion
			fmt.Print("What is your height (inches portion)? ")
			var inches float64
			_, err = fmt.Scan(&inches)
			if err != nil {
				fmt.Printf("Error reading inches: %v. Please try again.", err)
				continue
			}

			// Get height in inches.
			height = feetInchesToInches(feet, inches)
		default:
			return 0, fmt.Errorf("Invalid measurement system: %s", system)
		}

		break
	}

	return height, nil
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
func promptAge() (a string) {
	fmt.Print("Enter age: ")
	fmt.Scanln(&a)
	return a
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

// validateActivity validates their user response.
func validateActivity(a string) error {
	a = strings.ToLower(a)
	_, err := activity(a)
	if err != nil {
		return err
	}

	return nil
}

// PrintUserInfo prints the users info.
func PrintUserInfo(u *UserInfo) {
	fmt.Println(colorUnderline, "User Information:", colorReset)
	fmt.Printf("Measurement System: %s\n", u.System)
	fmt.Printf("Sex: %s\n", u.Sex)

	switch u.System {
	case "metric":
		fmt.Printf("Weight: %.2f kg\n", lbsToKg(u.Weight))
		fmt.Printf("Height: %.2f cm\n", inchesToCm(u.Height))
	case "imperial":
		feet, inches := inchesToFeetInches(u.Height)
		fmt.Printf("Weight: %.2f lbs\n", u.Weight)
		fmt.Printf("Height: %d' %.2f\"\n", feet, inches)
	default:
		fmt.Println("Invalid measurement system.")
	}

	fmt.Printf("Age: %d\n", u.Age)
	fmt.Printf("Activity Level: %s\n", u.ActivityLevel)
	fmt.Printf("TDEE: %.2f\n", u.TDEE)
}

// UpdateUserInfo lets the user update their information.
func UpdateUserInfo(u *UserInfo) {
	fmt.Println("Update your information.")
	getUserInfo(u)

	// Update min and max values for macros.
	setMinMaxMacros(u)

	// Update suggested macro split.
	protein, carbs, fats := calculateMacros(u)
	u.Macros.Protein = protein
	u.Macros.Carbs = carbs
	u.Macros.Fats = fats

	// Save the update UserInfo to config file.
	err := saveUserInfo(u)
	if err != nil {
		log.Printf("Failed to save user info: %v\n", err)
		return
	}

	fmt.Println("Updated information:")
	PrintUserInfo(u)
}
