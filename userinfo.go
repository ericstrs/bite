package bite

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
)

const (
	weightCol     = 0 // Dataframe column for weight.
	calsCol       = 1 // Dataframe column for calories.
	dateCol       = 2 // Dataframe column for weight.
	calsInProtein = 4 // Calories per gram of protein.
	calsInCarbs   = 4 // Calories per gram of carbohydrate.
	calsInFats    = 9 // Calories per gram of fat.
)

type UserInfo struct {
	UserID        int       `db:"user_id"`
	Sex           string    `db:"sex"`
	Weight        float64   `db:"weight"` // lbs
	Height        float64   `db:"height"` // cm
	Age           int       `db:"age"`
	ActivityLevel string    `db:"activity_level"`
	TDEE          float64   `db:"tdee"`
	Macros        Macros    `db:"macros"`
	MacrosID      int       `db:"macros_id"`
	System        string    `db:"system"`
	Phase         PhaseInfo `db:"phase"`
	PhaseID       int       `db:"phase_id"`
}

type Macros struct {
	MacrosID   int     `db:"macros_id"`
	Protein    float64 `db:"protein"`
	MinProtein float64 `db:"min_protein"`
	MaxProtein float64 `db:"max_protein"`
	Carbs      float64 `db:"carbs"`
	MinCarbs   float64 `db:"min_carbs"`
	MaxCarbs   float64 `db:"max_carbs"`
	Fats       float64 `db:"fats"`
	MinFats    float64 `db:"min_fats"`
	MaxFats    float64 `db:"max_fats"`
}

// ReadConfig reads user info from the SQLite database
func ReadConfig(db *sqlx.DB) (*UserInfo, error) {
	// Start a new transaction.
	tx, err := db.Beginx()
	if err != nil {
		return &UserInfo{}, err
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	u := &UserInfo{}

	// Query the database for the configuration (assuming only one
	// config for now).
	err = tx.Get(u, "SELECT * FROM config LIMIT 1")
	if err != nil {
		// If no config data in the database, generate a new config.
		if err == sql.ErrNoRows {
			u, err = generateAndSaveConfig(tx)
			if err != nil {
				return nil, err
			}

			return u, tx.Commit()
		}
		log.Printf("Error: Can't fetch config: %v\n", err)
		return nil, err
	}

	// Fetch related data from the macros and phase_info tables.
	macros, err := getMacros(tx, u.MacrosID)
	if err != nil {
		log.Printf("Error: Can't fetch macros: %v\n", err)
		return nil, err
	}
	u.Macros = *macros

	phase, err := getPhaseInfo(tx, u.PhaseID)
	if err != nil {
		log.Printf("Error: Can't fetch phase_info: %v\n", err)
		return nil, err
	}
	u.Phase = *phase

	log.Println("Loaded config.")
	return u, tx.Commit()
}

// generateAndSaveConfig generates a new config file for the user.
func generateAndSaveConfig(tx *sqlx.Tx) (*UserInfo, error) {
	fmt.Println("Please provide required information:")

	u := UserInfo{}

	// Get user details.
	getUserInfo(&u)

	processUserInfo(&u)

	// Save user info.
	err := saveUserInfo(tx, &u)
	if err != nil {
		log.Println("Failed to save user info:", err)
		return nil, err
	}

	return &u, nil
}

// getMacros fetches data from the macros table using the macros_id.
func getMacros(tx *sqlx.Tx, macrosID int) (*Macros, error) {
	m := &Macros{}
	err := tx.Get(m, "SELECT * FROM macros WHERE macros_id = ?", macrosID)
	return m, err
}

// getPhaseInfo fetches data from the phase_info table using the phase_id.
func getPhaseInfo(tx *sqlx.Tx, phaseID int) (*PhaseInfo, error) {
	p := &PhaseInfo{}
	err := tx.Get(p, "SELECT * FROM phase_info WHERE phase_id = ?", phaseID)
	return p, err
}

// saveUserInfo takes a transaction and user information and stores it
// in the database. It breaks down the task into separate functions for
// clarity and maintainability.
func saveUserInfo(tx *sqlx.Tx, u *UserInfo) error {
	// Insert or update macro nutritional data related to the user.
	if err := insertOrUpdateMacros(tx, u); err != nil {
		return err
	}

	// Insert or update phase information related to the user's current
	// diet phase.
	if err := insertOrUpdatePhaseInfo(tx, u); err != nil {
		return err
	}

	// Insert or update general user information.
	if err := insertOrUpdateUserInfo(tx, u); err != nil {
		return err
	}

	return nil
}

// insertOrUpdateUserInfo attempts to insert a new user information
// record into the database. If a record for the user already exists,
// it updates the existing record with new data.
//
// Note: the strange nature of this function comes from the avoidance of
// creating a user table. In such case, the user id would come from matching
// record to hashed password.
func insertOrUpdateUserInfo(tx *sqlx.Tx, u *UserInfo) error {
	// Check if the record already exists
	var count int
	err := tx.Get(&count, "SELECT COUNT(*) FROM config WHERE user_id = 1")
	if err != nil {
		return err
	}

	if count == 0 {
		// Insert if no record found
		_, err = tx.Exec(`
        INSERT INTO config(user_id, sex, weight, height, age, activity_level, tdee, system, macros_id, phase_id)
        VALUES (1, $1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			u.Sex, u.Weight, u.Height, u.Age, u.ActivityLevel, u.TDEE, u.System, u.Macros.MacrosID, u.Phase.PhaseID)

		if err != nil {
			log.Printf("Failed to insert into config table: %v\n", err)
		}
		return err
	}

	// Update if record found
	_, err = tx.Exec(`
			UPDATE config SET
					sex = $1, weight = $2, height = $3, age = $4,
					activity_level = $5, tdee = $6, system = $7, macros_id = $8, phase_id = $9
			WHERE user_id = 1`,
		u.Sex, u.Weight, u.Height, u.Age, u.ActivityLevel, u.TDEE, u.System, u.Macros.MacrosID, u.Phase.PhaseID)

	if err != nil {
		log.Printf("Failed to update into config table: %v\n", err)
	}
	return err
}

// insertOrUpdateMacros attempts to insert new macro nutritional data
// for the user. If a record for the user's macros already exists, it
// updates the existing record.
//
// Note: the strange nature of this function comes from the avoidance of
// creating a user table. In such case, the user id would come from matching
// record to hashed password.
func insertOrUpdateMacros(tx *sqlx.Tx, u *UserInfo) error {
	const macrosID = 1 // Constant ID value for macros

	_, err := tx.Exec(`
        INSERT INTO macros(macros_id, protein, min_protein, max_protein, carbs,
													min_carbs, max_carbs, fats, min_fats, max_fats)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
        ON CONFLICT(macros_id)
        DO UPDATE SET
            protein = $2, min_protein = $3, max_protein = $4,
            carbs = $5, min_carbs = $6, max_carbs = $7,
            fats = $8, min_fats = $9, max_fats = $10`,
		macrosID, u.Macros.Protein, u.Macros.MinProtein, u.Macros.MaxProtein,
		u.Macros.Carbs, u.Macros.MinCarbs, u.Macros.MaxCarbs,
		u.Macros.Fats, u.Macros.MinFats, u.Macros.MaxFats)
	if err != nil {
		return err
	}

	// update the UserInfo struct
	u.Macros.MacrosID = macrosID

	return err
}

// insertOrUpdatePhaseInfo attempts to insert new phase record for the
// user. If a record already exists, it updates the existing record.
//
// Note: the strange nature of this function comes from the avoidance of
// creating a user table. In such case, the user id would come from matching
// record to hashed password.
func insertOrUpdatePhaseInfo(tx *sqlx.Tx, u *UserInfo) error {
	// Check if there's an existing active phase for this user
	var existingPhaseID int
	err := tx.Get(&existingPhaseID, "SELECT phase_id FROM phase_info WHERE user_id = $1 AND status = 'active'", u.UserID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	// If active phase found, update it
	if err != sql.ErrNoRows {
		// Update the existing active phase
		_, err = tx.Exec(`
      UPDATE phase_info SET
        name = $2, goal_calories = $3, start_weight = $4, goal_weight = $5,
        weight_change_threshold = $6, weekly_change = $7, start_date = $8,
        end_date = $9, last_checked_week = $10, duration = $11,
        max_duration = $12, min_duration = $13, status = $14
        WHERE phase_id = $1`,
			existingPhaseID, u.Phase.Name, u.Phase.GoalCalories, u.Phase.StartWeight, u.Phase.GoalWeight,
			u.Phase.WeightChangeThreshold, u.Phase.WeeklyChange, u.Phase.StartDate.Format(dateFormat),
			u.Phase.EndDate.Format(dateFormat), u.Phase.LastCheckedWeek.Format(dateFormat), u.Phase.Duration,
			u.Phase.MaxDuration, u.Phase.MinDuration, u.Phase.Status)
		if err != nil {
			return err
		}

		return nil
	}
	// Otherwise, Insert a new phase
	res, err := tx.Exec(`
      INSERT INTO phase_info(user_id, name, status, goal_calories, start_weight, goal_weight,
        weight_change_threshold, weekly_change, start_date,
        end_date, last_checked_week, duration, max_duration,
        min_duration, status)
      VALUES ($1, $2, 'active', $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		u.UserID, u.Phase.Name, u.Phase.GoalCalories, u.Phase.StartWeight, u.Phase.GoalWeight,
		u.Phase.WeightChangeThreshold, u.Phase.WeeklyChange, u.Phase.StartDate.Format(dateFormat),
		u.Phase.EndDate.Format(dateFormat), u.Phase.LastCheckedWeek.Format(dateFormat), u.Phase.Duration,
		u.Phase.MaxDuration, u.Phase.MinDuration, u.Phase.Status)
	if err != nil {
		return err
	}

	phaseID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	// update the UserInfo struct
	u.Phase.PhaseID = int(phaseID)

	return nil
}

// updatePhaseInfo updates the user's ongoing phase details.
func updatePhaseInfo(tx *sqlx.Tx, u *UserInfo) error {
	// Check if there's an existing active phase for this user
	var activePhaseID int
	err := tx.Get(&activePhaseID, "SELECT phase_id FROM phase_info WHERE user_id = $1 AND status = 'active' LIMIT 1", u.UserID)
	if err != nil && err != sql.ErrNoRows {
		// If no active phase found, return error
		if err == sql.ErrNoRows {
			log.Printf("Could not find an active diet phase: %v\n", err)
			return err
		}
		return err
	}

	// Update the existing active phase
	_, err = tx.Exec(`
      UPDATE phase_info SET
        name = $2, goal_calories = $3, start_weight = $4, goal_weight = $5,
        weight_change_threshold = $6, weekly_change = $7, start_date = $8,
        end_date = $9, last_checked_week = $10, duration = $11,
        max_duration = $12, min_duration = $13, status = $14
        WHERE phase_id = $1`,
		activePhaseID, u.Phase.Name, u.Phase.GoalCalories, u.Phase.StartWeight, u.Phase.GoalWeight,
		u.Phase.WeightChangeThreshold, u.Phase.WeeklyChange, u.Phase.StartDate.Format(dateFormat),
		u.Phase.EndDate.Format(dateFormat), u.Phase.LastCheckedWeek.Format(dateFormat), u.Phase.Duration,
		u.Phase.MaxDuration, u.Phase.MinDuration, u.Phase.Status)
	if err != nil {
		log.Println("Error updating diet phase information.")
		return err
	}

	return nil
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

	// If fat calculation is less than minimum allowed fats,
	if u.Macros.MinFats > fats {
		fmt.Println("Fats are below minimum limit. Taking calories from carbs and moving them to fats.")
		// move calories from carbs to fats in an attempt to reach minimum
		// fat.

		fatsNeeded := u.Macros.MinFats - fats

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
	}

	// If fat caculation is greater than maximum allowed fats.
	if u.Macros.MaxFats < fats {
		// move calories from carbs and add to fats to reach maximum fats.

		fmt.Println("Calculated fats are above maximum amount. Taking calories from fats and moving them to carbs.")

		fatsToRemove := fats - u.Macros.MaxFats

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
func UpdateUserInfo(db *sqlx.DB, u *UserInfo) error {
	// Start a new transaction.
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	fmt.Println("Update your information.")
	getUserInfo(u)

	// Update min and max values for macros.
	setMinMaxMacros(u)

	// Update suggested macro split.
	protein, carbs, fats := calculateMacros(u)
	u.Macros.Protein = protein
	u.Macros.Carbs = carbs
	u.Macros.Fats = fats

	// Save the updated UserInfo.
	err = saveUserInfo(tx, u)
	if err != nil {
		log.Printf("Failed to save user info: %v\n", err)
		return err
	}

	fmt.Println("Updated information:")
	PrintUserInfo(u)

	return tx.Commit()
}
