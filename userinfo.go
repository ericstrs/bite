package calories

import (
	"fmt"
	"io/ioutil"
	"math"

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
	Weight        float64   `yaml:"weight"`
	Height        float64   `yaml:"height"`
	Age           int       `yaml:"age"`
	ActivityLevel string    `yaml:"activity_level"`
	TDEE          float64   `yaml:"tdee"`
	Macros        Macros    `taml:"macros"`
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

// lbsToKg converts pounts to kilograms.
func lbsToKg(w float64) float64 {
	return w * 0.45359237
}

// Mifflin calculates and returns the Basal Metabolic Rate (BMR) which is
// based on weight (kg), height (cm), age (years), and sex.
func Mifflin(u *UserInfo) float64 {
	// Convert weight from pounds to kilograms.
	weight := lbsToKg(u.Weight)

	factor := 5
	if u.Sex == "female" {
		factor = -151
	}

	var bmr float64
	bmr = (10 * weight) + (6.25 * u.Height) - (5 * float64(u.Age)) + float64(factor)
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
// This function prioritizes protein. Once optimal protein and carbs
// have been calculated, fat is then calculated and checked for minimum
// value. If fat is below the minimum limit, calories carbs are first to
// be moved (followed by protein) in an attempt to reach the minimum
// fat limit. This results in a possibly unbalanced macro split where
// fats and carbs are at their minimum, but protein sits at its optimal
// value.
func calculateMacros(u *UserInfo) (float64, float64, float64) {
	// Get the calories from the minimum macro values.
	mc := getMacroCals(u.Macros.MinProtein, u.Macros.MinCarbs, u.Macros.MinFats)
	if mc > u.Phase.GoalCalories {
		// Let the user know their daily calories are too low and update
		// daily calories to the minimum allowed.
		fmt.Printf("Minimum macro values in calories exceed original calorie goal of %.2f\n", u.Phase.GoalCalories)

		// Update phase daily goal calories.
		u.Phase.GoalCalories = mc
		fmt.Println("New daily calorie goal:", u.Phase.GoalCalories)

		return u.Macros.MinProtein, u.Macros.MinCarbs, u.Macros.MinFats
	}

	// Get the calories from the maximum macro values.
	mc = getMacroCals(u.Macros.MaxProtein, u.Macros.MaxCarbs, u.Macros.MaxFats)
	if mc < u.Phase.GoalCalories {
		// Let the user know their daily calories are too high and update
		// daily calories to the maximum allowed.
		fmt.Printf("Maximum macro values exceed original calorie goal of %f\n", u.Phase.GoalCalories)

		// Update phase daily goal calories.
		u.Phase.GoalCalories = mc
		fmt.Println("New daily calorie goal:", u.Phase.GoalCalories)

		return u.Macros.MaxProtein, u.Macros.MaxCarbs, u.Macros.MaxFats
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

	// If fat caculation is less than minimum allowed fats.
	if u.Macros.MinFats > fats {
		fmt.Println("Fats are below minimum limit. Taking calories from carbs and moving them to fats.")
		// Get some calories from carbs and add to fats to reach minimum.

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
		// Otherwise, we have reached the each minimum carb and protein
		// limit.

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

		fmt.Println("Calculated fats are above maximum amount. Taking calories from fats and moving them to carbs.")
		// Get some calories from carbs and add to fats to reach maximum.

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
