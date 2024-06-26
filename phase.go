// bite package provides dietary insight.
package bite

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type WeightGainStatus int
type WeightLossStatus int
type WeightMaintenanceStatus int

const (
	calsPerPound                                       = 3500 // Estimated calories per pound of bodyweight.
	lostTooLittle              WeightLossStatus        = -1
	withinLossRange            WeightLossStatus        = 0
	lostTooMuch                WeightLossStatus        = 1
	lost                       WeightMaintenanceStatus = -1
	maintained                 WeightMaintenanceStatus = 0
	gained                     WeightMaintenanceStatus = 1
	gainedTooLittle            WeightGainStatus        = -1
	withinGainRange            WeightGainStatus        = 0
	gainedTooMuch              WeightGainStatus        = 1
	minEntriesPerWeek                                  = 2
	defaultCutDuration                                 = 8.0    // Weeks.
	defaultBulkDuration                                = 10.0   // Weeks.
	defaultCutWeeklyChangePct                          = -0.005 // -0.5% of bodyweight per week.
	defaultBulkWeeklyChangePct                         = 0.0025 // +0.25% of bodyweight per week.
	dateFormat                                         = "2006-01-02"
	colorReset                                         = "\033[0m"
	colorItalic                                        = "\033[3m"
	colorRed                                           = "\033[31m"
	colorGreen                                         = "\033[32m"
	colorUnderline                                     = "\033[4m"
)

type PhaseInfo struct {
	PhaseID      int     `db:"phase_id"`
	UserID       int     `db:"user_id"`
	Name         string  `db:"name"`
	GoalCalories float64 `db:"goal_calories"`
	StartWeight  float64 `db:"start_weight"`
	GoalWeight   float64 `db:"goal_weight"`
	// WeightChangeThreshold is used to ensure the user has not
	// lost/gained too much weight for a given diet phase.
	// If the user chooses to continue the current diet phase,
	// WeightChangeThreshold is updated to the 10% of the user's current
	// weight, and the process repeats.
	WeightChangeThreshold float64   `db:"weight_change_threshold"`
	WeeklyChange          float64   `db:"weekly_change"`
	StartDate             time.Time `db:"start_date"`
	EndDate               time.Time `db:"end_date"`
	// First day of the most recent valid week.
	LastCheckedWeek time.Time `db:"last_checked_week"`
	Duration        float64   `db:"duration"`
	MaxDuration     float64   `db:"max_duration"`
	MinDuration     float64   `db:"min_duration"`
	Status          string    `db:"status"`
}

// CheckProgress performs checks on the user's current diet phase.
//
// Current solution to defining a week is continually adding 7 days to
// the start date. Weeks are only considered that contain at least two
// two entries for a given week.
func CheckProgress(db *sqlx.DB, u *UserInfo, entries *[]Entry) error {
	// Start a new transaction.
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	// Make a map to track the numbers of entries in each week.
	entryCountPerWeek, err := countEntriesPerWeek(u, entries)
	if err != nil {
		return err
	}

	// Count number of valid weeks.
	validWeeks := countValidWeeks(*entryCountPerWeek)

	// If less than 2 valid weeks after the diet start date,
	// then do nothing, and return early.
	if validWeeks < 2 {
		log.Println("There is less than 2 weeks of entries after the diet start date. Skipping remaining checks on user progress.")
		return nil
	}

	switch u.Phase.Name {
	case "cut":
		var total float64
		var status WeightLossStatus

		err := checkCutThreshold(tx, u) // Ensure user hasn't lost too much weight.
		if err != nil {
			return err
		}

		status, total, err = checkCutLoss(tx, u, entries) // Ensure weekly weight loss.
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
		status, total, err := checkMaintenance(tx, u, entries) // Ensure maintenance.
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
		var total float64
		var status WeightGainStatus

		err := checkBulkThreshold(tx, u) // Ensure user hasn't gained too much weight.
		if err != nil {
			return err
		}

		status, total, err = checkBulkGain(tx, u, entries) // Ensure weekly weight gain.
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

	return tx.Commit()
}

// countEntriesPerWeek returns a map to tracker the number of entires in
// each weeks of a diet phase.
func countEntriesPerWeek(u *UserInfo, entries *[]Entry) (*map[int]int, error) {
	entryCountPerWeek := make(map[int]int)
	weekNumber := 0

	// Get the first day and the upcoming Sunday
	firstDay := u.Phase.StartDate
	firstSunday := firstDay.AddDate(0, 0, (int)(7-firstDay.Weekday())%7)

	// Count entries in the first (partial) week
	entryCount, err := countEntriesInWeek(entries, firstDay, firstSunday)
	if err != nil {
		return nil, err
	}
	entryCountPerWeek[weekNumber] = entryCount
	weekNumber++

	// For subsequent weeks,
	for date := firstSunday.AddDate(0, 0, 1); date.Before(u.Phase.EndDate) || isSameDay(date, u.Phase.EndDate); date = date.AddDate(0, 0, 7) {
		weekStart := date
		weekEnd := date.AddDate(0, 0, 6)

		// Count the number of entries within the current week.
		entryCount, err := countEntriesInWeek(entries, weekStart, weekEnd)
		if err != nil {
			return nil, err
		}
		entryCountPerWeek[weekNumber] = entryCount
		weekNumber++
	}
	return &entryCountPerWeek, nil
}

// countEntriesInWeek finds the number of entires within a given week.
func countEntriesInWeek(entries *[]Entry, weekStart, weekEnd time.Time) (int, error) {
	count := 0
	// Starting from the start date, iterate over the days of the week,
	// and count the number of logged days.
	for i := weekStart; i.Before(weekEnd) || isSameDay(i, weekEnd); i = i.AddDate(0, 0, 1) {
		idx, err := findEntryIdx(entries, i)
		if err != nil {
			return -1, err
		}

		// If entry exists with `i` date,
		if idx != -1 {
			count++
		}
	}

	return count, nil
}

// countValidWeeks counts ands returns the number of valid weeks in a
// given diet phase. A strict definition of a valid week need not be
// performed here. This is because its only used to ensure the diet has
// `minEntriesPerWeek` entries.
func countValidWeeks(e map[int]int) int {
	count := 0
	for week := 0; week < len(e); week++ {
		if e[week] > minEntriesPerWeek {
			count++
		}
	}

	return count
}

// checkCutThreshold checks if the user has lost too much weight, in
// which the user is presented with different options of moving forward.
//
// Assumptions:
// * User has lost some amount of weight.
// * `u.Phase.WeightChangeThreshold` has been initialized.
func checkCutThreshold(tx *sqlx.Tx, u *UserInfo) error {
	// If user has gained more weight than starting weight, return early.
	if u.Weight > u.Phase.StartWeight {
		return nil
	}

	// Find the amount of weight the user has lost.
	weightLost := u.Phase.StartWeight - u.Weight

	// If the user has lost more than the threshold weight change,
	if weightLost > u.Phase.WeightChangeThreshold {
		option := getCutAction()

		switch option {
		case "1":
			// Update current diet phase status to: "completed".
			u.Phase.Status = "completed"
			if err := updatePhaseInfo(tx, u); err != nil {
				return err
			}

			if err := transitionToMaintenance(tx, u); err != nil {
				return err
			}
		case "2": // Change to different phase.
			// Update current diet phase status to: "stopped".
			u.Phase.Status = "stopped"
			if err := updatePhaseInfo(tx, u); err != nil {
				return err
			}

			if err := processPhaseTransition(tx, u); err != nil {
				return err
			}
		case "3": // Continue with the cut.
			u.Phase.WeightChangeThreshold += u.Weight * 0.10 // 10% of current weight.
			// Save user info to config file.
			err := saveUserInfo(tx, u)
			if err != nil {
				log.Println("Failed to update phase start weight:", err)
				return err
			}
			percentage := (u.Phase.WeightChangeThreshold / u.Phase.StartWeight) * 100.0
			fmt.Printf("Maximum threshold updated to %.1f%% of starting weight, you can continue with the cut.\n", percentage)
		}
	}

	return nil
}

// getCutAction prompts user for the action given that they've
// already surpassed cut threshold, validates their reponse
// until they've entered a valid action, and returns the valid action.
func getCutAction() string {
	fmt.Println("You've reached the maximum threshold for weight loss (you've lost more than 10%% of your starting weight in a single cutting phase). Stopping your cut and beginning a maintenance phase is highly recommended. Please choose one of the following actions:")
	fmt.Println("1. End cut and begin maintenance phase")
	fmt.Println("2. Choose a different diet phase.")
	fmt.Println("3. Continue with the cut.")

	var option string
	for {
		option = promptAction()

		err := validateAction(option)
		if err != nil {
			fmt.Println("Invalid action. Please try again.")
			continue
		}

		break
	}
	return option
}

// checkCutLoss checks to see if user is on the track to meeting weight
// loss goal.
func checkCutLoss(tx *sqlx.Tx, u *UserInfo, entries *[]Entry) (WeightLossStatus, float64, error) {
	weeksUnderGoal := 0 // Consecutive weeks where the user gained too much weight.
	weeksOverGoal := 0  // Consecutive weeks where the user gained too little weight.
	totalLossUnderGoal := 0.0
	totalLossOverGoal := 0.0

	resetCounters := func() {
		weeksUnderGoal = 0
		weeksOverGoal = 0
		totalLossUnderGoal = 0
		totalLossOverGoal = 0
	}

	// Iterate over each week of the diet.
	for date := u.Phase.LastCheckedWeek; date.Before(u.Phase.EndDate); date = date.AddDate(0, 0, 7) {
		weekStart := date
		weekEnd := date.AddDate(0, 0, 6)

		valid, totalWeekWeightChange, _, err := validWeek(tx, entries, weekStart, weekEnd, u)
		if err != nil {
			return 0, 0, err
		}

		if !valid {
			resetCounters()
			continue
		}

		status := metWeeklyGoalCut(u, totalWeekWeightChange)

		switch status {
		case lostTooLittle:
			weeksUnderGoal++
			totalLossUnderGoal += totalWeekWeightChange
			weeksOverGoal = 0
			totalLossOverGoal = 0
		case lostTooMuch:
			weeksOverGoal++
			totalLossOverGoal += totalWeekWeightChange
			weeksUnderGoal = 0
			totalLossUnderGoal = 0
		case withinLossRange:
			resetCounters()
		}

		if weeksUnderGoal >= 2 {
			return status, totalLossUnderGoal, nil
		}

		if weeksOverGoal >= 2 {
			return status, totalLossOverGoal, nil
		}
	}

	return withinLossRange, 0, nil
}

// validWeek determines if a given week fits the definition of a
// week, retrives total change in weight, and array of calories for
// the given week.
func validWeek(tx *sqlx.Tx, entries *[]Entry, weekStart, weekEnd time.Time, u *UserInfo) (bool, float64, []float64, error) {
	// Does this week contain has at least `minEntriesPerWeek` entries?
	entryCount, err := countEntriesInWeek(entries, weekStart, weekEnd)
	if err != nil || entryCount < minEntriesPerWeek {
		return false, 0, nil, err
	}

	// Does `weekStart` fall within the diet phase?
	totalWeekWeightChange, valid, err := totalWeightChangeWeek(entries, weekStart, weekEnd, u)
	if err != nil || !valid {
		return false, 0, nil, err
	}

	// Get array of calories for given week.
	dailyCalories, err := getCalsWeek(entries, weekStart, weekEnd)
	if err != nil {
		log.Println(err)
		return false, 0, nil, err
	}

	// Did the user adhere to the daily calorie goal for this week?
	valid = metWeeklyCalGoal(u, dailyCalories)
	if !valid {
		return false, 0, nil, nil
	}

	// Once the week has passed all the checks, update the last checked
	// week in the diet phase to the last day of the week.
	u.Phase.LastCheckedWeek = weekEnd

	// Save the updated last checked week to config file.
	err = saveUserInfo(tx, u)
	if err != nil {
		log.Printf("Failed to save user info: %v\n", err)
		return false, 0, nil, err
	}
	log.Println("Updated last checked week to:", weekEnd)

	return true, totalWeekWeightChange, dailyCalories, nil
}

// getCalsWeek returns an float64 array containing calorie count for
// each day in a given week.
//
// Assumptions:
// * Given week has at least `minEntriesPerWeek` entries.
func getCalsWeek(entries *[]Entry, weekStart, WeekEnd time.Time) ([]float64, error) {
	var calsWeek []float64

	// Get the dataframe index of the entry with the start date of the
	// diet.
	startIdx, err := findEntryIdx(entries, weekStart)
	if err != nil {
		return nil, err
	}
	if startIdx == -1 {
		return nil, fmt.Errorf("ERROR: No matching entry for date %s\n", weekStart)
	}

	endIdx := min(startIdx+7, len(*entries))

	// If there were less than `minEntriesPerWeek` entries found in this
	// week, then return early.
	if endIdx-startIdx < minEntriesPerWeek {
		log.Printf("Given week has less than %d entries.\n", minEntriesPerWeek)
		return nil, fmt.Errorf("ERROR: Given week has less than %d entries.\n", minEntriesPerWeek)
	}

	// Iterate over each day of the week starting from startIdx.
	for i := startIdx; i < endIdx; i++ {
		// Get entry calories.
		cal := (*entries)[i].Calories
		calsWeek = append(calsWeek, cal) // Append recorded daily calorie.
	}

	return calsWeek, nil
}

// metWeeklyCalGoal calculates whether the user met their daily calorie
// goal on at least 70% of the days in the week.
func metWeeklyCalGoal(u *UserInfo, dailyCalories []float64) bool {
	daysMetGoal := 0
	for _, cal := range dailyCalories {
		if metCalDayGoal(u, cal) {
			daysMetGoal++
		}
	}
	// Compare to 70% of the week.
	return float64(daysMetGoal)/float64(len(dailyCalories)) >= 0.7
}

// metWeeklyGoalCut checks to see if a given week has met the weekly
// change in weight goal
func metWeeklyGoalCut(u *UserInfo, totalWeekWeightChange float64) WeightLossStatus {
	lowerTolerance := u.Phase.WeeklyChange * 0.2
	upperTolerance := math.Abs(u.Phase.WeeklyChange) * 0.1

	// If user did not lose enough this week,
	if totalWeekWeightChange > u.Phase.WeeklyChange+upperTolerance {
		return lostTooLittle
	}
	// If user lost too much this week,
	if totalWeekWeightChange < u.Phase.WeeklyChange+lowerTolerance {
		return lostTooMuch
	}

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

// checkBulkThreshold checks if the user has gained too much weight, in
// which the bulk is stopped and a maintenance phase begins.
//
// Assumptions:
// * User has gained some amount of weight.
// * `u.Phase.WeightChangeThreshold` has been initialized.
func checkBulkThreshold(tx *sqlx.Tx, u *UserInfo) error {
	// If user has lost more weight than starting weight, return early.
	if u.Weight < u.Phase.StartWeight {
		return nil
	}

	// Find the amount of weight the user has gained.
	weightGain := u.Weight - u.Phase.StartWeight

	// If the user has gained more than 10% of starting weight,
	if weightGain > u.Phase.WeightChangeThreshold {
		option := getBulkAction()
		switch option {
		case "1": // User wants to transition to maintenance phase.
			// Update current diet phase status to: "completed".
			u.Phase.Status = "completed"
			if err := updatePhaseInfo(tx, u); err != nil {
				return err
			}

			fmt.Println("Going from a bulk to a maintenance phase, you should gradually drop your calories down to your TDEE.")
			if err := transitionToMaintenance(tx, u); err != nil {
				return nil
			}
		case "2": // User want to change to different phase.
			// Update current diet phase status to: "stopped".
			u.Phase.Status = "stopped"
			if err := updatePhaseInfo(tx, u); err != nil {
				return err
			}

			if err := processPhaseTransition(tx, u); err != nil {
				return err
			}
		case "3": // User wants to continue with the bulk.
			u.Phase.WeightChangeThreshold += u.Weight * 0.10 // 10% of current weight.
			// Save user info to config file.
			err := saveUserInfo(tx, u)
			if err != nil {
				log.Println("Failed to update phase start weight:", err)
				return err
			}
			percentage := (u.Phase.WeightChangeThreshold / u.Phase.StartWeight) * 100.0
			fmt.Printf("Maximum threshold updated to %.1f%% of starting weight, continuing with the bulk.\n", percentage)
		}
	}

	return nil
}

// transitionToMaintenance starts a new maintienance phase.
// Note: diet duration is left unmodified so the maintenance phase
// lasts as long as the previous diet phase.
func transitionToMaintenance(tx *sqlx.Tx, u *UserInfo) error {
	u.Phase.Name = "maintain"
	u.Phase.GoalCalories = u.TDEE
	u.Phase.StartWeight = u.Weight
	u.Phase.WeightChangeThreshold = 0
	u.Phase.WeeklyChange = 0
	u.Phase.GoalWeight = u.Phase.StartWeight
	u.Phase.LastCheckedWeek = u.Phase.StartDate
	u.Phase.Status = "active"
	u.Phase.StartDate = time.Now()
	u.Phase.EndDate = calculateEndDate(u.Phase.StartDate, u.Phase.Duration)
	setMinMaxPhaseDuration(u)
	promptConfirmation(u)

	// Save user info to config file.
	err := saveUserInfo(tx, u)
	if err != nil {
		log.Println("Failed to update phase to maintenance:", err)
		return err
	}
	fmt.Println("Saved user information.")
	return nil
}

// getBulkAction prompts user for the action given that they've
// already surpassed bulk thresholds, validates their reponse
// until they've entered a valid action, and returns the valid action.
func getBulkAction() string {
	fmt.Println("You've reached the maximum threshold for weight gain (you've gained more than 10%% of your starting weight in a single bulking phase). Stopping your bulk and beginning a maintenance phase is highly recommended. Please choose one of the following actions:")
	fmt.Println("1. End bulk and begin maintenance phase")
	fmt.Println("2. Choose a different diet phase.")
	fmt.Println("3. Continue with the bulk.")

	var option string
	for {

		option = promptAction()

		err := validateAction(option)
		if err != nil {
			fmt.Println("Invalid action. Please try again.")
			continue
		}

		break
	}
	return option
}

// promptAction prompts the user for the action.
func promptAction() (o string) {
	fmt.Printf("Type number and <Enter>: ")
	fmt.Scanln(&o)
	return o
}

// validateAction validates and returns the user action.
func validateAction(option string) error {
	if option == "1" || option == "2" || option == "3" {
		return nil
	}

	return errors.New("Invalid action.")
}

// CheckPhaseStatus checks if the phase is active.
func CheckPhaseStatus(db *sqlx.DB, u *UserInfo) (string, error) {
	// Start a new transaction.
	tx, err := db.Beginx()
	if err != nil {
		return "", err
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	t := time.Now()
	// If today comes before diet start date, then phase has not yet begun.
	if t.Before(u.Phase.StartDate) {
		log.Println("Diet phase has not yet started. Skipping check on diet phase.")
		return "", nil
	}

	// If today comes after diet end date, diet phase is over.
	if t.After(u.Phase.EndDate) {
		fmt.Println("Diet phase completed! Starting the diet phase transistion process.")
		//  Update current diet phase status to: "completed".
		u.Phase.Status = "completed"
		if err := updatePhaseInfo(tx, u); err != nil {
			return "", err
		}

		// Process phase transition
		if err := processPhaseTransition(tx, u); err != nil {
			return "", err
		}

		return u.Phase.Status, tx.Commit()
	}

	// If today is the first day of the diet (active status has yet to be
	// updated),
	if u.Phase.Status == "scheduled" {
		// Set starting weight to current weight.
		u.Phase.StartWeight = u.Weight

		// Check if goal weight is still valid.
		_, err := validateGoalWeight(strconv.FormatFloat(u.Phase.GoalWeight, 'f', -1, 64), u)
		// If weight is now invalid,
		if err != nil {
			option := getNextAction(u)

			switch option {
			case "1": // Get new goal weight.
				u.Phase.GoalWeight = getGoalWeight(u)
			case "2": // Change to different phase.
				// Update current diet phase status to: "stopped".
				u.Phase.Status = "stopped"
				if err := updatePhaseInfo(tx, u); err != nil {
					return "", err
				}

				if err := processPhaseTransition(tx, u); err != nil {
					return "", err
				}
			}
		}
		u.Phase.Status = "active"
	}
	return u.Phase.Status, tx.Commit()
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
		if err := validateNextAction(a); err != nil {
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
func promptNextAction() (a string) {
	fmt.Printf("Type number and <Enter>: ")
	fmt.Scanln(&a)
	return a
}

// validateNextAction validates the next action.
func validateNextAction(a string) error {
	if a == "1" || a == "2" {
		return nil
	}

	return errors.New("Invalid action.")
}

// checkMaintenance ensures user is maintaining the same weight.
func checkMaintenance(tx *sqlx.Tx, u *UserInfo, entries *[]Entry) (WeightMaintenanceStatus, float64, error) {
	weeksGained := 0 // Consecutive weeks where the user gained too much weight.
	weeksLost := 0   // Consecutive weeks where the user lost too much weight.
	totalGain := 0.0
	totalLoss := 0.0

	resetCounters := func() {
		weeksGained = 0
		weeksLost = 0
		totalGain = 0
		totalLoss = 0
	}

	// Iterate over each week of the diet.
	for date := u.Phase.LastCheckedWeek; date.Before(u.Phase.EndDate); date = date.AddDate(0, 0, 7) {
		weekStart := date
		weekEnd := date.AddDate(0, 0, 6)

		valid, totalWeekWeightChange, _, err := validWeek(tx, entries, weekStart, weekEnd, u)
		if err != nil {
			return 0, 0, err
		}

		if !valid {
			resetCounters()
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
			resetCounters()
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
		return lost
	}
	// If user gained too much weight this week,
	if totalWeekWeightChange > u.Phase.WeeklyChange+upperTolerance {
		return gained
	}

	return maintained
}

// checkBulkGain checks to see if user is on the track to meeting weight
// gain goal.
func checkBulkGain(tx *sqlx.Tx, u *UserInfo, entries *[]Entry) (WeightGainStatus, float64, error) {
	weeksUnderGoal := 0 // Consecutive weeks where the user gained too much weight.
	weeksOverGoal := 0  // Consecutive weeks where the user gained too little weight.
	totalGainUnderGoal := 0.0
	totalGainOverGoal := 0.0

	resetCounters := func() {
		weeksUnderGoal = 0
		weeksOverGoal = 0
		totalGainUnderGoal = 0
		totalGainOverGoal = 0
	}

	// Iterate over each week of the diet.
	for date := u.Phase.LastCheckedWeek; date.Before(u.Phase.EndDate); date = date.AddDate(0, 0, 7) {
		weekStart := date
		weekEnd := date.AddDate(0, 0, 6)

		valid, totalWeekWeightChange, _, err := validWeek(tx, entries, weekStart, weekEnd, u)
		if err != nil {
			return 0, 0, err
		}

		if !valid {
			resetCounters()
			continue
		}

		status := metWeeklyGoalBulk(u, totalWeekWeightChange)

		switch status {
		case gainedTooLittle:
			weeksUnderGoal++
			totalGainUnderGoal += totalWeekWeightChange
			weeksOverGoal = 0
			totalGainOverGoal = 0
		case gainedTooMuch:
			weeksOverGoal++
			totalGainOverGoal += totalWeekWeightChange
			weeksUnderGoal = 0
			totalGainUnderGoal = 0
		case withinGainRange:
			resetCounters()
		}

		if weeksUnderGoal >= 2 {
			return status, totalGainUnderGoal, nil
		}

		if weeksOverGoal >= 2 {
			return status, totalGainOverGoal, nil
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
		return gainedTooLittle
	}
	// If user gained too much this week,
	if totalWeekWeightChange > u.Phase.WeeklyChange+upperTolerance {
		return gainedTooMuch
	}

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
//
// Assumptions:
// * The given week has been checked for minEntriesPerWeek.
func totalWeightChangeWeek(entries *[]Entry, weekStart, weekEnd time.Time, u *UserInfo) (float64, bool, error) {
	totalWeightChangeWeek := 0.0

	// Get the dataframe index of the entry with the start date of the
	// diet.
	startIdx, err := findEntryIdx(entries, weekStart)
	if err != nil || startIdx == -1 {
		return 0, false, err
	}

	endIdx := min(startIdx+7, len(*entries))

	// If there were zero entries found in the week, then return early.
	if endIdx-startIdx < minEntriesPerWeek {
		log.Printf("Less than %d entries found this week.\n", minEntriesPerWeek)
		return 0, false, fmt.Errorf("ERROR: Less than %d entries found this week.\n", minEntriesPerWeek)
	}

	// Iterate over each day of the week starting from startIdx.
	for i := startIdx; i < endIdx; i++ {
		// Get entry date.
		date := (*entries)[i].Date

		// If date falls after the end of the week, break out of loop.
		if date.After(weekEnd) {
			break
		}

		// Get entry weight.
		weight := (*entries)[i].UserWeight

		// Get the previous weight to current day.
		previousWeight, err := getPrecedingWeightToDay(u, entries, weight, i)
		if err != nil {
			return 0, false, err
		}

		// Calculate the weight change between two days.
		weightChange := weight - previousWeight

		// Update total weight change
		totalWeightChangeWeek += weightChange
	}

	return totalWeightChangeWeek, true, nil
}

// min finds and returns the smaller integer.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// findEntryIdx finds the index of an entry given a date.
func findEntryIdx(entries *[]Entry, d time.Time) (int, error) {
	for i, entry := range *entries {
		date := entry.Date
		if isSameDay(date, d) {
			return i, nil
		}
		continue
	}

	return -1, nil
}

// getPrecedingWeightToDay returns the preceding entry to a given week.
func getPrecedingWeightToDay(u *UserInfo, entries *[]Entry, weight float64, startIdx int) (float64, error) {
	var previousWeight float64
	// If entry is the first in the dataframe, set previous weight
	// equal to zero. This is necessary to prevent index out of bounds
	// error.
	if startIdx == 0 {
		previousWeight = weight
		return previousWeight, nil
	}

	//pw := logs.Series[weightCol].Value(startIdx - 1).(string)
	// Get previous entry's weight.
	previousWeight = (*entries)[startIdx-1].UserWeight

	return previousWeight, nil
}

// processPhaseTransition transitions the user to a new diet phase
// and saves the next phase to config file.
func processPhaseTransition(tx *sqlx.Tx, u *UserInfo) error {
	fmt.Println("Step 1: Diet phase recap")
	fmt.Printf("Goal weight: %f. Current weight: %f\n", u.Phase.GoalWeight, u.Weight)

	printTransitionSuggestion(u.Phase.Name)

	processUserInfo(u)

	// Save user info to config file.
	err := saveUserInfo(tx, u)
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

	// Set min and max diet phase duration.
	setMinMaxPhaseDuration(u)

	// Set initial diet start weight.
	u.Phase.StartWeight = u.Weight

	// Set initial diet weight change theshold.
	u.Phase.WeightChangeThreshold = u.Weight * 0.10

	promptUserForPhaseInfo(u)

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

// promptUserForPhaseInfo prompts user for information to initialize the Phase
// struct, validates their response until they enter valid values, and
// sets their value to the corresponding struct field. Some fields are
// simply calculated.
func promptUserForPhaseInfo(u *UserInfo) {
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
		fmt.Printf("Lose 0.5%% of bodyweight per week for 8 weeks.\n")
	case "maintain":
		fmt.Printf("Maintain same weight for 5 weeks.\n")
	case "bulk":
		fmt.Printf("Gain 0.25%% of bodyweight per week for 10 weeks.\n")
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
	c = strings.ToLower(c)
	if c == "recommended" || c == "custom" {
		return nil
	}

	return errors.New("Invalid diet choice.")
}

// handleRecommendedDiet sets UserInfo struct fields according to a
// reccomended diet.
func handleRecommendedDiet(u *UserInfo) {
	u.Phase.StartDate = getStartDate(u)

	switch u.Phase.Name {
	case "cut":
		goalWeight, dailyCaloricChange := calculateDietPlan(u.Phase.StartWeight, defaultCutDuration, defaultCutWeeklyChangePct)
		setRecommendedValues(u, defaultCutWeeklyChangePct*u.Phase.StartWeight, defaultCutDuration, goalWeight, u.TDEE+dailyCaloricChange)
	case "maintain":
		setRecommendedValues(u, 0, 5, u.Phase.StartWeight, u.TDEE)
	case "bulk":
		goalWeight, dailyCaloricChange := calculateDietPlan(u.Phase.StartWeight, defaultBulkDuration, defaultBulkWeeklyChangePct)
		setRecommendedValues(u, defaultBulkWeeklyChangePct*u.Phase.StartWeight, defaultBulkDuration, goalWeight, u.TDEE+dailyCaloricChange)
	}

	u.Phase.EndDate = calculateEndDate(u.Phase.StartDate, u.Phase.Duration)
}

// calculateDietPlan calculates the goal weight and daily caloric change needed
// to achieve the goal weight in the given duration.
func calculateDietPlan(startWeight, duration, weeklyChangePct float64) (goalWeight, dailyCaloricChange float64) {
	goalWeight = calculateGoalWeight(startWeight, duration, weeklyChangePct)
	totalWeekWeightChangeCals := weeklyChangePct * startWeight * calsPerPound
	dailyCaloricChange = totalWeekWeightChangeCals / 7.0
	return goalWeight, dailyCaloricChange
}

// calculateGoalWeight calculates the estimated goal weight for a given
// diet phase.
func calculateGoalWeight(startWeight, duration, weeklyChange float64) float64 {
	// Start with the current weight.
	currentWeight := startWeight

	// Calculate total weeks and remaining days.
	totalWeeks := int(duration)
	remainingDays := duration - float64(totalWeeks)

	// For each full week in the diet phase.
	for i := 0; i < totalWeeks; i++ {
		// Calculate the change for this week as a percentage of the
		// current weight.
		changeThisWeek := currentWeight * weeklyChange

		// Add this week's change to the current weight.
		currentWeight += changeThisWeek
	}

	// If there are remaining days in the last week.
	if remainingDays > 0 {
		// Calculate the change for remaining days as a proportion of
		// the weekly change.
		changeRemainingDays := (currentWeight * weeklyChange) * (remainingDays / 7.0)

		// Add remaining days' change to the current weight.
		currentWeight += changeRemainingDays
	}

	// Return the final expected weight
	return math.Round(currentWeight*100) / 100
}

// setRecommendedValues sets the recommended values for the UserInfo
// fields: weekly weight change, diet duration, diet goal weight, and
// diet daily calories.
func setRecommendedValues(u *UserInfo, w, d, g, c float64) {
	u.Phase.WeeklyChange = w
	u.Phase.Duration = d
	u.Phase.GoalWeight = g
	u.Phase.GoalCalories = c
	u.Phase.LastCheckedWeek = u.Phase.StartDate
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

	// Initialize last checked week.
	u.Phase.LastCheckedWeek = u.Phase.StartDate

	// set diet end date.
	setEndDate(u)

	// Get diet goal weight.
	u.Phase.GoalWeight = getGoalWeight(u)

	// Calculate weekly weight change rate.
	u.Phase.WeeklyChange = calculateWeeklyChange(u.Weight, u.Phase.GoalWeight, u.Phase.Duration)

	// Get weekly average weight change in calories.
	totalWeekWeightChangeCals := u.Phase.WeeklyChange * calsPerPound
	// Calculate daily average weight change in caloric needed for cut or bulk.
	avgDayWeightChangeCals := totalWeekWeightChangeCals / 7

	switch u.Phase.Name {
	case "cut":
		u.Phase.GoalCalories = u.TDEE - avgDayWeightChangeCals
	case "maintain":
		u.Phase.GoalCalories = u.TDEE
	case "bulk":
		u.Phase.GoalCalories = u.TDEE + avgDayWeightChangeCals
	}
}

// getStartDate prompts user for diet start date, validates user response
// until user enters valid date, and returns valid date.
func getStartDate(u *UserInfo) (date time.Time) {
	for {
		// Prompt user for diet start date.
		r := promptDate("Enter diet start date (YYYY-MM-DD) [Press Enter for today's date]: ")

		u.Phase.Status = "scheduled" // set active to scheduled by default.
		// If user entered default date,
		if r == "" {
			// set date to today's date.
			r = time.Now().Format(dateFormat)
			// Set phase status to true.
			u.Phase.Status = "active"
		}

		// Ensure user response is a date.
		var err error
		date, err = ValidateDateStr(r)
		if err != nil {
			fmt.Printf("%v. Please try again.\n", err)
			continue
		}

		// Ensure date is not in the past.
		if !validateDateIsNotPast(date) {
			fmt.Println("Date must be today or future date. Please try again.")
			continue
		}

		break
	}
	return date
}

// validateDateIsNotPast validates a given date to ensure it does not
// fall in the past. The function returns `true` if the input date is
// equal to or later than the current date (today) and `false`
// otherwise.
func validateDateIsNotPast(date time.Time) bool {
	today := time.Now()
	if date.After(today) || isSameDay(date, today) {
		return true // Date is today or later
	}
	return false // Date is earlier than today
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

// promptDate prompts and returns diet date.
func promptDate(promptStr string) string {
	reader := bufio.NewReader(os.Stdin)
	// Prompt user for diet date.
	fmt.Printf("%s ", promptStr)
	response, _ := reader.ReadString('\n')

	// Trim leading/trailing white space (including newlines)
	response = strings.TrimSpace(response)

	return response
}

// validateEndDate prompts user for diet end date, validates user
// response, and returns diet end date and diet duration when date is
// valid.
func validateEndDate(r string, u *UserInfo) (time.Time, float64, error) {
	// Ensure user response is a date.
	d, err := ValidateDateStr(r)
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

// ValidateDateStr validates the given date string and returns date if
// valid.
func ValidateDateStr(dateStr string) (time.Time, error) {
	// Validate user response.
	date, err := time.Parse(dateFormat, dateStr)
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
	fmt.Println("Diet Start Date:", u.Phase.StartDate.Format(dateFormat))
	fmt.Println("Diet End Date:", u.Phase.EndDate.Format(dateFormat))
	fmt.Printf("Diet Duration: %.1f weeks\n", math.Round(u.Phase.Duration*100)/100)

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
	s = strings.ToLower(s)
	// If user response is either "cut", "maintain", or "bulk",
	if s == "cut" || s == "maintain" || s == "bulk" {
		return nil
	}

	return errors.New("Invalid diet phase.")
}

// Assumptions:
// * Diet phase activity has been checked. That is, this function should
// not be called for a diet phase that is not currently active.
func Summary(u *UserInfo, entries *[]Entry) {
	defer printDietPhaseInfo(u)

	m, _ := countEntriesPerWeek(u, entries)
	totalEntries := 0
	totalWeeks := 0
	for _, entryCount := range *m {
		totalEntries += entryCount
		totalWeeks++
	}

	// Check if there are any days logged for this diet.
	if totalEntries == 0 {
		log.Println("There has yet to be a logged day for this diet phase. Skipping diet day summary.")
		return
	}

	daySummary(u, entries)

	if totalWeeks < 1 {
		log.Println("There has yet to be a logged week for this diet phase. Skipping diet week summary.")
		return
	}

	weekSummary(u, entries)

	if totalWeeks < 4 {
		log.Println("There has yet to be a logged month for this diet phase. Skipping diet month summary.")
		return
	}

	monthSummary(u, entries)
}

// daySummary prints a summary of the diet for the current day.
func daySummary(u *UserInfo, entries *[]Entry) {
	today := time.Now()
	i := len(*entries) - 1

	// Get most recent entry date.
	tailDate := (*entries)[i].Date

	// Ensure most recent entry date is equal to today's date.
	if !isSameDay(today, tailDate) {
		fmt.Println("Missing entry for today. Please create today's entry prior to attempting to generate today's diet summary.")
		return
	}

	cals := (*entries)[i].Calories

	fmt.Printf("%sDay Summary for %s%s\n", colorUnderline, tailDate.Format(dateFormat), colorReset)
	fmt.Printf("Current Weight: %.2f\n", u.Weight)
	fmt.Printf("Calories Consumed: ")
	c := getAdherenceColor(fmt.Sprintf("%.2f", cals), metCalDayGoal(u, cals))
	fmt.Printf("%s\n", c)
}

// metCalDayGoal checks to see if the user met the daily calorie goal
// given their current diet phase.
func metCalDayGoal(u *UserInfo, cals float64) bool {
	tolerance := 0.05 * u.Phase.GoalCalories

	switch u.Phase.Name {
	case "cut":
		return cals <= u.Phase.GoalCalories
	case "bulk":
		return cals >= u.Phase.GoalCalories
	case "maintain":
		return math.Abs(cals-u.Phase.GoalCalories) <= tolerance
	default:
		return false
	}
}

// getAdherenceColor returns some text in either green or red
// indicating whether or not user adhered to the diet caloire goal for a
// particular day.
func getAdherenceColor(s string, b bool) string {
	switch b {
	case true:
		return colorGreen + s + colorReset
	case false:
		return colorRed + s + colorReset
	default:
		return ""
	}
}

// weekSummary prints a summary of the diet for the most recent week.
func weekSummary(u *UserInfo, entries *[]Entry) {
	fmt.Println()
	fmt.Println(colorUnderline, "Week Summary", colorReset)

	var daysOfWeek []string
	var calsOfWeek []string
	//var calsStr string
	today := time.Now()

	// Find the current ISO week.
	_, currentWeek := today.ISOWeek()

	//tailDate, _ := time.Parse(dateFormat, logs.Series[dateCol].Value(logs.NRows()-1).(string))

	i := len(*entries) - 1
	// Find the most recent entry's date.
	tailDate := (*entries)[i].Date

	// Find the last Monday that comes before tailDate
	diff := (int(tailDate.Weekday()-time.Monday+6)%7 + 1) % 7
	lastMonday := tailDate.AddDate(0, 0, -diff)

	// Find the tail ISO week.
	_, tailWeek := lastMonday.ISOWeek()

	// Ensure tail week is equal to this week.
	if tailWeek != currentWeek {
		fmt.Println("Missing entries for this week. Please create today's entry prior to attempting to generate this week's diet summary.")
		return
	}

	// Iterate over the entries starting from EndDate - 7 days.
	for i := 0; i < 7; i++ {
		date := lastMonday.AddDate(0, 0, i)
		d := date.Weekday().String() + " "

		// Bold the value if it's the current day.
		if date.Equal(tailDate) {
			d = colorItalic + date.Weekday().String() + colorReset + " "
		}

		// Append date in day of the week to array.
		daysOfWeek = append(daysOfWeek, d)

		idx, _ := findEntryIdx(entries, date)
		// If date matches a logged entry date,
		if idx != -1 {
			cals := (*entries)[idx].Calories
			s := getAdherenceColor(fmt.Sprintf("%-10.2f", cals), metCalDayGoal(u, cals))

			calsOfWeek = append(calsOfWeek, s)

			continue
		}
		calsOfWeek = append(calsOfWeek, "")
	}

	printWeekSummary(daysOfWeek, calsOfWeek)
}

// monthSummary prints a summary of the diet for the most recent 4 weeks.
func monthSummary(u *UserInfo, entries *[]Entry) {
	fmt.Println()
	fmt.Println(colorUnderline, "Month Summary", colorReset)
	today := time.Now()

	currentYear, currentMonth, _ := today.Date()

	i := len(*entries) - 1
	// Find the most recent entry's date.
	tailDate := (*entries)[i].Date

	// Find the last Monday that comes before tailDate
	diff := (int(tailDate.Weekday()-time.Monday+6)%7 + 1) % 7
	lastMonday := tailDate.AddDate(0, 0, -diff)

	tailYear, tailMonth, _ := lastMonday.Date()

	// Ensure tail month is equal to this month.
	// If tailMonth is not equal to the current month or tailYear is not
	// the current year, then don't print the summary
	if tailMonth != currentMonth || tailYear != currentYear {
		fmt.Println("Missing entries for this month. Please create today's entry prior to attempting to generate this month's diet summary.")
		return
	}

	// Iterate over the weeks starting from EndDate - 28 days.
	for week := 0; week < 4; week++ {
		weekStart := lastMonday.AddDate(0, 0, -21+week*7)

		var daysOfWeek []string
		var calsOfWeek []string

		// Iterate over the days of the week.
		for i := 0; i < 7; i++ {
			date := weekStart.AddDate(0, 0, i)
			d := date.Weekday().String()

			// Bold the value if it's the current day.
			if date.Equal(tailDate) {
				d = colorItalic + date.Weekday().String() + colorReset + " "
			}
			// Append date in day of the week to array.
			daysOfWeek = append(daysOfWeek, d)

			idx, _ := findEntryIdx(entries, date)
			// If date matches a logged entry date,
			if idx != -1 {
				cals := (*entries)[idx].Calories
				s := getAdherenceColor(fmt.Sprintf("%-10.2f", cals), metCalDayGoal(u, cals))

				calsOfWeek = append(calsOfWeek, s)

				continue
			}
			calsOfWeek = append(calsOfWeek, "")
		}

		printWeekSummary(daysOfWeek, calsOfWeek)
	}
}

// printWeekSummary prints a summary of the diet for a week.
func printWeekSummary(daysOfWeek []string, calsOfWeek []string) {
	for _, day := range daysOfWeek {
		fmt.Printf("%-10s", day)
	}
	fmt.Println()

	for _, cal := range calsOfWeek {
		fmt.Printf("%-10s", cal)
	}
	fmt.Println()
}

// isSameDay checks to see if two dates have the same year, month, and
// day.
func isSameDay(date1, date2 time.Time) bool {
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// printDietPhaseInfo prints out the information about the diet phase.
func printDietPhaseInfo(u *UserInfo) {
	// Print the diet phase information.
	fmt.Println()
	fmt.Println(colorUnderline, "Diet Phase Info:", colorReset)
	fmt.Println("Diet phase:", u.Phase.Name)
	fmt.Println("Start Date:", u.Phase.StartDate.Format(dateFormat))
	fmt.Println("End Date:", u.Phase.EndDate.Format(dateFormat))
	fmt.Printf("Duration: %.1f weeks\n", math.Round(u.Phase.Duration*100)/100)

	remainingTime := calculateDuration(time.Now(), u.Phase.EndDate)
	remainingDays := int(remainingTime.Hours() / 24)
	fmt.Printf("Remaining time: %d days\n", remainingDays)

	fmt.Println("Goal Weight:", u.Phase.GoalWeight)
	fmt.Println("Start Weight:", u.Phase.StartWeight)
}

// StopPhase stops the ongoing diet and prompts the user for
// information on the next phase.
func StopPhase(db *sqlx.DB, u *UserInfo) error {
	// Start a new transaction.
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	// Update current diet phase status to: "stopped".
	u.Phase.Status = "stopped"
	if err := updatePhaseInfo(tx, u); err != nil {
		return err
	}

	if err := processPhaseTransition(tx, u); err != nil {
		return err
	}

	return tx.Commit()
}
