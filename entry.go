package bite

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

const (
	SearchLimit       = 100
	weightSearchLimit = 10
	dateFormatTime    = "15:04:05"
	fullBlock         = "\u2588"
	lightBlock        = "\u2592"

	// PortionSize sets the portion size for a food. It should be noted
	// that the recorded serving size in the foods table does not align
	// with the recorded nutrients for the same food. For all foods, the
	// nutrients amount correspond to serving size of 100.
	PortionSize = 100
)

var ErrDone = errors.New("done")

// Entry fields will be constructed from daily_weights and daily_foods
// table during runtime.
type Entry struct {
	UserWeight float64   `db:"user_weight"`
	Calories   float64   `db:"calories"`
	Date       time.Time `db:"date"`
	Protein    float64   `db:"protein"`
	Carbs      float64   `db:"carbs"`
	Fat        float64   `db:"fat"`
	Price      float64   `db:"price"`
}

type WeightEntry struct {
	ID     int       `db:"id"`
	Date   time.Time `db:"date"`
	Weight float64   `db:"weight"`
}

type DailyFood struct {
	ID               int       `db:"id"`
	FoodName         string    `db:"food_name"`
	FoodID           int       `db:"food_id"`
	MealID           *int      `db:"meal_id"`
	Date             time.Time `db:"date"`
	ServingSize      float64   `db:"serving_size"`
	ServingUnit      string    `db:"serving_unit"`
	NumberOfServings float64   `db:"number_of_servings"`
	Calories         float64   `db:"calories"`
	Price            float64   `db:"price"`
	FoodMacros       *FoodMacros
}

type DailyFoodCount struct {
	DailyFood
	Count int `db:"count"`
}

// AllEntries returns all the user's entries from the database.
func AllEntries(db *sqlx.DB) (*[]Entry, error) {
	query := `
	SELECT
		dw.date,
		dw.weight AS user_weight,
		SUM(df.calories) AS calories,
		SUM(df.protein) AS protein,
		SUM(df.carbs) AS carbs,
		SUM(df.fat) AS fat
	FROM daily_weights dw
	JOIN daily_foods df ON dw.date = df.date
	GROUP BY dw.date, dw.weight
	ORDER BY dw.date
	`

	var entries []Entry
	err := db.Select(&entries, query)
	if err != nil {
		log.Fatalf("AllEntries: %v\n", err)
	}

	return &entries, nil
}

// PrintEntries prints given slice of entries.
func PrintEntries(entries []Entry) {
	fmt.Println("-------------------------------------------------------------------------")
	fmt.Println("| Date       | Weight      | Calories | Protein (g) | Carbs (g) | Fat (g) |")
	fmt.Println("-------------------------------------------------------------------------")
	for _, entry := range entries {
		dateStr := entry.Date.Format("2006-01-02")
		fmt.Printf("| %-10s | %-12.2f | %-8.2f | %-11.2f | %-9.2f | %-7.2f |\n", dateStr, entry.UserWeight, entry.Calories, entry.Protein, entry.Carbs, entry.Fat)
	}
	fmt.Println("-------------------------------------------------------------------------")
}

// LogWeight gets weight and date from user to create a new weight entry.
func LogWeight(u *UserInfo, db *sqlx.DB) error {
	// Start a new transaction
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	for {
		// Get weight from user
		weight, err := getWeight(u.System)
		if err != nil {
			fmt.Printf("%v. Please try again.\n", err)
			continue
		}

		// Get weight entry date from user
		date := promptDateNotPast("Enter weight entry date")

		if err = addWeightEntry(tx, date, weight); err != nil {
			fmt.Printf("%v. Please try again.\n", err)
			continue
		}

		// Update users weight
		u.Weight = weight
		if err := insertOrUpdateUserInfo(tx, u); err != nil {
			return err
		}
		break
	}

	return tx.Commit()
}

// addWeightEntry inserts a weight entry into the database.
func addWeightEntry(tx *sqlx.Tx, date time.Time, weight float64) error {
	// Ensure weight hasn't already been logged for given date.
	exists, err := checkWeightExists(tx, date)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("Weight for this date has already been logged.")
	}

	// Insert the new weight entry into the weight database.
	_, err = tx.Exec(`INSERT INTO daily_weights (date, time, weight) VALUES ($1, $2, $3)`, date.Format(dateFormat), date.Format(dateFormatTime), weight)
	if err != nil {
		return err
	}

	fmt.Println("Successfully added weight entry.")
	return nil
}

// promptDateNotPast prompts user for date that it not in the past, validates user
// response until user enters a valid date, and return the valid date.
func promptDateNotPast(s string) (date time.Time) {
	for {
		// Prompt user for diet start date.
		r := promptDate(fmt.Sprintf("%s (YYYY-MM-DD) [Press <Enter> for today's date]: ", s))

		// If user entered default date,
		if r == "" {
			// set date to today's date.
			r = time.Now().Format(dateFormat)
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

// ShowWeightLog prints entire weight log.
func ShowWeightLog(db *sqlx.DB) error {
	log, err := allWeightEntries(db)
	if err != nil {
		return err
	}
	printWeightEntries(log)
	return nil
}

// UpdateWeightLog updates the weight value for a given weight log.
func UpdateWeightLog(db *sqlx.DB, u *UserInfo) error {
	// Let user select weight entry to update.
	entry, err := selectWeightEntry(db)
	if err != nil {
		return err
	}

	// Get new weight.
	weight, err := getWeight(u.System)

	if err := updateWeightEntry(db, entry.ID, weight); err != nil {
		return err
	}
	fmt.Println("Updated weight entry.")

	return nil
}

// updateWeightEntry performs the database update operation.
//
// Assumptions:
// * Weight id exists in the database table.
func updateWeightEntry(db *sqlx.DB, id int, newWeight float64) error {
	const updateSQL = `
		UPDATE daily_weights
		SET weight = $1
		WHERE id = $2
`
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(updateSQL, newWeight, id); err != nil {
		return err
	}
	return tx.Commit()
}

// DeleteWeightEntry deletes a weight entry.
func DeleteWeightEntry(db *sqlx.DB) error {
	// Get selected weight entry.
	entry, err := selectWeightEntry(db)
	if err != nil {
		return err
	}
	if err := deleteOneWeightEntry(db, entry.ID); err != nil {
		return err
	}
	fmt.Println("Deleted weight entry.")
	return nil
}

// deleteOneWeightEntry deletes one weight entry from the database.
func deleteOneWeightEntry(db *sqlx.DB, id int) error {
	const deleteSQL = `
    DELETE FROM daily_weights
    WHERE id = $1
`
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(deleteSQL, id); err != nil {
		return err
	}
	return tx.Commit()
}

// selectWeightEntry prints the user's weight entries, prompts them to select
// a weight entry, and returns the selected weight entry.
func selectWeightEntry(db *sqlx.DB) (WeightEntry, error) {
	// Get all weight logs.
	entries, err := recentWeightEntries(db)
	if err != nil {
		return WeightEntry{}, err
	}

	// Print recent weight entries.
	printWeightEntries(entries)

	// Get response.
	response := promptSelectEntry("Enter entry index to select or date to search (YYYY-MM-DD): ")
	idx, err := strconv.Atoi(response)

	// While response is an integer
	for err == nil {
		// If integer is invalid,
		if 1 > idx || idx > len(entries) {
			fmt.Println("Number must be between 0 and number of entries. Please try again.")
			response = promptSelectEntry("Enter entry index to select or date to search (YYYY-MM-DD): ")
			idx, err = strconv.Atoi(response)
			continue
		}
		// Otherwise, return food at valid index.
		return entries[idx-1], nil
	}
	// User response was a date to search.

	// While user response is not an integer,
	for {
		// Validate user response.
		date, err := ValidateDateStr(response)
		if err != nil {
			fmt.Printf("%v. Please try again.", err)
			response = promptSelectEntry("Enter entry index to select or date to search (YYYY-MM-DD): ")
			continue
		}

		// Get the filtered entries.
		entry, err := searchWeightLog(db, date)
		if err != nil {
			return WeightEntry{}, err
		}

		// If no match found,
		if entry == nil {
			fmt.Println("No match found. Please try again.")
			response = promptSelectEntry("Enter entry index to select or date to search (YYYY-MM-DD): ")
			continue
		}

		// Print entry.
		fmt.Printf("[1] %s %f\n", entry.Date.Format(dateFormat), entry.Weight)

		response = promptSelectEntry("Enter entry index to select or date to search (YYYY-MM-DD): ")
		idx, err := strconv.Atoi(response)

		// While response is an integer
		for err == nil {
			// If integer is invalid,
			if idx != 1 {
				fmt.Println("Number must be 1. Please try again.")
				response = promptSelectEntry("Enter entry index to select or date to search (YYYY-MM-DD): ")
				idx, err = strconv.Atoi(response)
				continue
			}
			// Otherwise, return entry at valid index.
			return *entry, nil
		}
		// User response was a search term. Continue to next loop.
	}
}

// printWeightEntries prints out specified weight entries.
func printWeightEntries(entries []WeightEntry) {
	for i, entry := range entries {
		fmt.Printf("[%d] %s %f\n", i+1, entry.Date.Format(dateFormat), entry.Weight)
	}
}

// allWeightEntries returns all the user's logged weight entries.
func allWeightEntries(db *sqlx.DB) ([]WeightEntry, error) {
	// Since DailyWeight struct does not currently support time field, the
	// queury excludes the time field from the selected records.
	const query = `
		SELECT id, date, weight FROM daily_weights ORDER by date DESC"
		`
	wl := []WeightEntry{}
	if err := db.Select(&wl, query); err != nil {
		return nil, err
	}
	return wl, nil
}

// recentWeightEntries returns the user's logged weight entries up to
// a limit.
func recentWeightEntries(db *sqlx.DB) ([]WeightEntry, error) {
	// Since DailyWeight struct does not currently support time field, the
	// queury excludes the time field from the selected records.
	const query = `
		SELECT id, date, weight FROM daily_weights
		ORDER BY date DESC
		LIMIT $1
		`
	wl := []WeightEntry{}
	if err := db.Select(&wl, query, weightSearchLimit); err != nil {
		return nil, err
	}
	return wl, nil
}

// promptSelectEntry prompts and returns entry to select or a search
// term.
func promptSelectEntry(s string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s: ", s)
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	// Remove the newline character at the end of the string
	response = strings.TrimSpace(response)
	return response
}

// searchWeightLog searches through all weight entries and returns the
// entry that matches the entered date.
func searchWeightLog(db *sqlx.DB, d time.Time) (*WeightEntry, error) {
	// Since DailyWeight struct does not currently support time field, the
	// queury excludes the time field from the selected records.
	const query = `
		SELECT id, date, weight FROM daily_weights
		ORDER by date = $1
		LIMIT 1
		`
	var entry WeightEntry
	if err := db.Get(&entry, query, d.Format(dateFormat)); err != nil {
		log.Printf("Search for weight entry failed: %v\n", err)
		return nil, err
	}

	return &entry, nil
}

// checkWeightExists checks if a weight entry already exists for the
// given date.
func checkWeightExists(tx *sqlx.Tx, date time.Time) (bool, error) {
	const query = `
   SELECT COUNT(*) FROM daily_weights
	 WHERE date = $1
	 `
	var count int
	if err := tx.Get(&count, query, date.Format(dateFormat)); err != nil {
		return false, err
	}
	return count > 0, nil
}

// LogFood lets the user log multiple foods.
func LogFood(db *sqlx.DB) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var selectedFoods []Food
	// While user wants to keep logging foods.
OuterLoop:
	for {
		food, err := selectFood(db)
		if err != nil {
			// If user has indicated they are done logging foods, then break
			if errors.Is(err, ErrDone) {
				break
			}
			log.Println(err)
			return err
		}

		// Get any existing preferences for the selected food.
		f, err := getFoodPref(tx, food.ID)
		if err != nil {
			return fmt.Errorf("couldn't get food preferences: %v", err)
		}

		// Display any existing preferences for the selected food.
		printFoodPref(*f)

		reader := bufio.NewReader(os.Stdin)
	UserInputLoop:
		for {
			fmt.Printf("What would you like to do? (1 = Update Values, 2 = Search Again) [Press <Enter> for Existing]: ")
			s, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Error reading input:", err)
				continue
			}
			s = strings.TrimSpace(s)

			switch s {
			case "": // User indicates they want to keep existing food preferences
				// Do nothing.
				break UserInputLoop
			case "1": // User indicates they want to change existing food preferences
				// Get updated food preferences.
				f = promptFoodPref(food.ID, f.ServingSize, f.NumberOfServings)
				// Make database update for food preferences.
				if err := UpdateFoodPrefs(tx, f); err != nil {
					return err
				}
				break UserInputLoop
			case "2": // User indicates they want to search again
				continue OuterLoop
			default:
				fmt.Println("Invalid choice. Please enter 1, 2, or press <Enter>.")
			}
		}

		// Get food with up to date food preferences.
		foodWithPref, err := FoodWithPref(db, food.ID)
		if err != nil {
			return err
		}

		selectedFoods = append(selectedFoods, *foodWithPref)
	}

	// When user indictes they are done before logging a single food,
	// return early.
	if len(selectedFoods) == 0 {
		fmt.Println("No food selected.")
		return nil
	}

	// Get date of food entry.
	date := promptDateNotPast("Enter food entry date")

	for _, f := range selectedFoods {
		// Log selected food to the food log database table. Taking into
		// account food preferences.
		if err := AddFoodEntry(tx, &f, date); err != nil {
			log.Println(err)
			return err
		}
	}

	fmt.Println("Successfully added food entry.")

	return tx.Commit()
}

// selectFood prompts the user to enter a search term, prints the matched
// foods, prompts user to enter an index to select a food or another
// serach term for a different food. This repeats until user enters a
// valid index.
func selectFood(db *sqlx.DB) (Food, error) {
	recentFoods, err := RecentlyLoggedFoods(db, SearchLimit)
	if err != nil {
		log.Println(err)
		return Food{}, err
	}

	fmt.Println("Recently logged foods:")
	for i, food := range recentFoods {
		fmt.Printf("[%d] %s\n", i+1, food.Name)
	}

	response := promptSelectEntry("Enter either food index, search term, or 'done'")
	idx, err := strconv.Atoi(response)

	// While response is an integer
	for err == nil {
		// If integer is invalid,
		if 1 > idx || idx > len(recentFoods) {
			fmt.Println("Number must be between 0 and number of entries. Please try again.")
			// Get response.
			response := promptSelectEntry("Enter either food index, search term, or 'done'")
			idx, err = strconv.Atoi(response)
			continue
		}
		return recentFoods[idx-1], nil
	}

	// If user enters "done", then return early.
	if response == "done" {
		return Food{}, ErrDone
	}

	// User response was a search term.

	// While user response is not an integer
	for {
		// Get filtered foods.
		filteredFoods, err := SearchFoods(db, response)
		if err != nil {
			return Food{}, fmt.Errorf("couldn't search for a food: %v", err)
		}

		// If no matches found,
		if len(filteredFoods) == 0 {
			fmt.Println("No matches found. Please try again.")
			response = promptSelectResponse("food")
			continue
		}

		// Print foods.
		for i, food := range filteredFoods {
			brandDetail := ""
			if food.BrandName != "" {
				brandDetail = " (Brand: " + food.BrandName + ")"
			}
			fmt.Printf("[%d] %s%s\n", i+1, food.Name, brandDetail)
		}

		response = promptSelectResponse("food")
		idx, err := strconv.Atoi(response)

		// While response is an integer
		for err == nil {
			// If integer is invalid,
			if 1 > idx || idx > len(filteredFoods) {
				fmt.Println("Number must be between 0 and number of foods. Please try again.")
				response = promptSelectResponse("food")
				idx, err = strconv.Atoi(response)
				continue
			}
			// Otherwise, return food at valid index.
			return (filteredFoods)[idx-1], nil
		}
		// User response was a search term. Continue to next loop.
	}
}

// RecentlyLoggedFoods retrieves most recently logged foods.
func RecentlyLoggedFoods(db *sqlx.DB, limit int) ([]Food, error) {
	const (
		allSQL = `
    SELECT f.*
    FROM (
	    SELECT *, ROW_NUMBER() OVER (PARTITION BY food_id ORDER BY date DESC) AS rn
	    FROM daily_foods
    ) AS df
    INNER JOIN foods f ON df.food_id = f.food_id
    WHERE df.rn = 1
    ORDER BY df.date DESC
    LIMIT $1
  `
		// Override existing serving size and number of servings if there
		// exists a matching entry in the food_prefs table for the food id.
		query = `
      SELECT
        COALESCE(fp.serving_size, f.serving_size, 100) AS serving_size,
        COALESCE(fp.number_of_servings, 1) AS number_of_servings
      FROM foods f
      LEFT JOIN food_prefs fp ON fp.food_id = f.food_id
      WHERE f.food_id = $1
      LIMIT 1
    `
		calSQL = `
      SELECT amount
      FROM food_nutrients
      WHERE food_id = $1 AND nutrient_id = 1008
      LIMIT 1
    `
	)

	var foods []Food
	if err := db.Select(&foods, allSQL, limit); err != nil {
		return nil, err
	}

	// For each matching food, find its serving size and number of
	// servings, calories, and macros. Taking into account any user
	// preferences for each food.
	for i := 0; i < len(foods); i++ {
		if err := db.Get(&foods[i], query, foods[i].ID); err != nil {
			return nil, fmt.Errorf("couldn't get serving size and number of servings for %q: %v", foods[i].Name, err)
		}

		if err := db.Get(&foods[i].Calories, calSQL, foods[i].ID); err != nil {
			return nil, fmt.Errorf("couldn't get portion calories for %q: %v", foods[i].Name, err)
		}
		var err error
		foods[i].FoodMacros = &FoodMacros{}
		foods[i].FoodMacros, err = foodMacros(db, foods[i].ID)
		if err != nil {
			return nil, fmt.Errorf("couldn't get macros for %q: %v", foods[i].Name, err)
		}

		ratio := foods[i].ServingSize / PortionSize
		foods[i].Calories *= ratio * foods[i].NumberOfServings
		foods[i].FoodMacros.Protein *= ratio * foods[i].NumberOfServings
		foods[i].FoodMacros.Fat *= ratio * foods[i].NumberOfServings
		foods[i].FoodMacros.Carbs *= ratio * foods[i].NumberOfServings
		foods[i].Price *= ratio * foods[i].NumberOfServings
	}

	return foods, nil
}

// SearchFoods searches through all foods and returns food that contain
// the search term. The matching foods have associated preferences,
// calorie, and macros.
func SearchFoods(db *sqlx.DB, term string) ([]Food, error) {
	const (
		searchSQL = `
			SELECT f.*
			FROM foods f
			INNER JOIN foods_fts s ON s.food_id = f.food_id
			WHERE foods_fts MATCH $1
			ORDER BY bm25(foods_fts)
			LIMIT $2`

		// Override existing serving size and number of servings if there
		// exists a matching entry in the food_prefs table for the food id.
		query = `
      SELECT
        COALESCE(fp.serving_size, f.serving_size, 100) AS serving_size,
        COALESCE(fp.number_of_servings, 1) AS number_of_servings
      FROM foods f
      LEFT JOIN food_prefs fp ON fp.food_id = f.food_id
      WHERE f.food_id = $1
      LIMIT 1
    `
		calSQL = `
			SELECT amount
			FROM food_nutrients
			WHERE food_id = $1 AND nutrient_id = 1008
			LIMIT 1
		`
	)
	foods := []Food{}

	// Get all matching foods.
	if err := db.Select(&foods, searchSQL, term, SearchLimit); err != nil {
		return nil, fmt.Errorf("couldn't get result foods: %v", err)
	}

	// For each matching food, find its serving size and number of
	// servings, calories, and macros. Taking into account any user
	// preferences for each food.
	for i := 0; i < len(foods); i++ {
		if err := db.Get(&foods[i], query, foods[i].ID); err != nil {
			return nil, fmt.Errorf("couldn't get serving size and number of servings for %q: %v", foods[i].Name, err)
		}

		if err := db.Get(&foods[i].Calories, calSQL, foods[i].ID); err != nil {
			return nil, fmt.Errorf("couldn't get portion calories for %q: %v", foods[i].Name, err)
		}
		var err error
		foods[i].FoodMacros = &FoodMacros{}
		foods[i].FoodMacros, err = foodMacros(db, foods[i].ID)
		if err != nil {
			return nil, fmt.Errorf("couldn't get macros for %q: %v", foods[i].Name, err)
		}

		ratio := foods[i].ServingSize / PortionSize
		foods[i].Calories *= ratio * foods[i].NumberOfServings
		foods[i].FoodMacros.Protein *= ratio * foods[i].NumberOfServings
		foods[i].FoodMacros.Fat *= ratio * foods[i].NumberOfServings
		foods[i].FoodMacros.Carbs *= ratio * foods[i].NumberOfServings
		foods[i].Price *= ratio * foods[i].NumberOfServings
	}

	return foods, nil
}

// promptSelectResponse prompts and returns meal to select or a search term.
func promptSelectResponse(item string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter either the index of the %s to select or a search term: ", item)
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("promptSelectResponse: %v\n", err)
	}
	// Remove the newline character at the end of the string
	response = strings.TrimSpace(response)
	return response
}

// getFoodPref gets the food preferences for the given food.
func getFoodPref(tx *sqlx.Tx, foodID int) (*FoodPref, error) {
	const query = `
	SELECT
		f.food_id,
		f.serving_size AS default_serving_size,
		COALESCE(fp.serving_size, f.serving_size, 100) AS serving_size,
		f.household_serving,
		COALESCE(fp.number_of_servings, 1) AS number_of_servings,
		f.serving_unit
	FROM foods f
	LEFT JOIN food_prefs fp ON f.food_id = fp.food_id
	WHERE f.food_id = $1
`
	var pref FoodPref
	if err := tx.Get(&pref, query, foodID); err != nil {
		if err == sql.ErrNoRows {
			return &FoodPref{}, fmt.Errorf("no preference found for food ID %d", foodID)
		}
		return &FoodPref{}, fmt.Errorf("unable to execute query: %w", err)
	}
	return &pref, nil
}

// printFoodPref prints the preferences for a food.
func printFoodPref(pref FoodPref) {
	fmt.Printf("Current Serving Size: %.2f %s\n", pref.ServingSize, pref.ServingUnit)
	fmt.Printf("Number of Servings: %.1f\n", pref.NumberOfServings)
}

// promptFoodPref prompts user for food preferences, validates their
// response until they've entered a valid response, and returns the
// valid response.
func promptFoodPref(foodID int, servingSize, numOfServings float64) *FoodPref {
	pref := &FoodPref{}
	pref.FoodID = foodID
	pref.ServingSize = promptUpdateServingSize(servingSize)
	pref.NumberOfServings = promptUpdateNumServings(numOfServings)
	return pref
}

// promptUpdateNumServings entered prints existing food number of
// serving and prompts user to enter a new one.
func promptUpdateNumServings(existingNumServings float64) float64 {
	var newNumServings string
	fmt.Printf("Current serving size: %.2f\n", existingNumServings)
	for {
		fmt.Printf("Enter new serving size [Press <Enter> to keep]: ")
		fmt.Scanln(&newNumServings)

		// User pressed <Enter>
		if newNumServings == "" {
			return existingNumServings
		}

		newNumServingsFloat, err := strconv.ParseFloat(newNumServings, 64)
		if err != nil || newNumServingsFloat < 0 {
			fmt.Println("Invalid float value entered. Please try again.")
			continue
		}
		return newNumServingsFloat
	}
}

// promptMealFoodPref prompts user for meal food preferences,
// validates their response until they've entered a valid response,
// and returns the valid response.
func promptMealFoodPref(foodID int, mealID int64, servingSize, numServings float64) *MealFoodPref {
	pref := &MealFoodPref{}
	pref.FoodID = foodID
	pref.MealID = mealID
	pref.ServingSize, _ = promptServingSize()
	pref.NumberOfServings = promptUpdateNumServings(numServings)
	return pref
}

// UpdateFoodPrefs updates the user's preferences for a given
// food.
func UpdateFoodPrefs(tx *sqlx.Tx, pref *FoodPref) error {
	const insertSQL = `
		INSERT INTO food_prefs (food_id, number_of_servings, serving_size)
		VALUES (:food_id, :number_of_servings, :serving_size)
		ON CONFLICT(food_id) DO UPDATE SET
		number_of_servings = :number_of_servings,
		serving_size = :serving_size
`
	if _, err := tx.NamedExec(insertSQL, pref); err != nil {
		log.Printf("Failed to update food prefs: %v\n", err)
		return err
	}
	return nil
}

// AddFoodEntry inserts a food entry into the database.
func AddFoodEntry(tx *sqlx.Tx, f *Food, date time.Time) error {
	const query = `
	INSERT INTO daily_foods (food_id, date, time, serving_size, number_of_servings, calories, protein, fat, carbs, price)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := tx.Exec(query, f.ID, date.Format(dateFormat), date.Format(dateFormatTime),
		f.ServingSize, f.NumberOfServings, f.Calories, f.FoodMacros.Protein,
		f.FoodMacros.Fat, f.FoodMacros.Carbs, f.Price)
	// If there was an error executing the query, return the error
	if err != nil {
		log.Println("Failed to insert food entry into daily_foods.")
		return err
	}
	return nil
}

// UpdateFoodLog updates an existing food entry in the database.
func UpdateFoodLog(db *sqlx.DB) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Let user select food entry to update.
	entry, err := selectFoodEntry(tx)
	if err != nil {
		return err
	}

	// Get new food preferences.
	pref := promptFoodPref(entry.FoodID, entry.ServingSize, entry.NumberOfServings)
	// Make database update for food preferences.
	if err := UpdateFoodPrefs(tx, pref); err != nil {
		return err
	}

	// Get food with up to date food preferences.
	foodWithPref, err := FoodWithPref(db, entry.FoodID)
	if err != nil {
		return err
	}
	// Update food entry.
	if err := updateFoodEntry(tx, entry.ID, *foodWithPref); err != nil {
		return err
	}

	fmt.Println("Updated food entry.")
	return tx.Commit()
}

// selectFoodEntry prints recently logged foods, prompts user to enter a
// search term, prompts user to enter an index to select a food entry or
// another search term for a different food entry. This repeats until
// user enters a valid index.
func selectFoodEntry(tx *sqlx.Tx) (DailyFood, error) {
	// Get most recently logged foods.
	recentFoods, err := recentFoodEntries(tx, SearchLimit)
	if err != nil {
		log.Println(err)
		return DailyFood{}, err
	}

	// Print recent food entries.
	printFoodEntries(recentFoods)

	// Get response.
	response := promptSelectEntry("Enter entry index to select or date to search (YYYY-MM-DD)")
	idx, err := strconv.Atoi(response)

	// While response is an integer
	for err == nil {
		// If integer is invalid,
		if 1 > idx || idx > len(recentFoods) {
			fmt.Println("Number must be between 0 and number of entries. Please try again.")
			response = promptSelectEntry("Enter entry index to select or date to search (YYYY-MM-DD)")
			idx, err = strconv.Atoi(response)
			continue
		}
		// Otherwise, return food at valid index.
		return recentFoods[idx-1], nil
	}
	// User response was a date to search.

	// While user response is a date,
	for {
		// Validate user response.
		date, err := ValidateDateStr(response)
		if err != nil {
			fmt.Printf("%v. Please try again.", err)
			response = promptSelectEntry("Enter entry index to select or date to search (YYYY-MM-DD)")
			continue
		}

		// Get the filtered entries.
		filteredEntries, err := searchFoodLog(tx, date)
		if err != nil {
			log.Println(err)
			return DailyFood{}, err
		}

		// If no matches found,
		if len(filteredEntries) == 0 {
			fmt.Println("No match found. Please try again.")
			response = promptSelectEntry("Enter entry index to select or date to search (YYYY-MM-DD)")
			continue
		}

		// Print the foods entries for given date.
		printFoodEntries(filteredEntries)

		response = promptSelectEntry("Enter entry index to select or date to search (YYYY-MM-DD)")
		idx, err := strconv.Atoi(response)

		// While response is an integer
		for err == nil {
			// If integer is invalid,
			if 1 > idx || idx > len(filteredEntries) {
				fmt.Println("Number must be between 0 and number of entries. Please try again.")
				response = promptSelectEntry("Enter entry index to select or date to search (YYYY-MM-DD)")
				idx, err = strconv.Atoi(response)
				continue
			}
			// Otherwise, return entry at valid index.
			return filteredEntries[idx-1], nil
		}
		// User response was a search term. Continue to next loop.
	}
}

// recentFoodEntries retrieves most recently logged food entries.
func recentFoodEntries(tx *sqlx.Tx, limit int) ([]DailyFood, error) {
	// Since DailyFood struct does not currently support time field, the
	// query excludes the time field from the selected records.
	const query = `
		SELECT df.id, df.food_id, df.meal_id, df.date, df.serving_size,
		df.number_of_servings, f.food_name, f.serving_unit
		FROM (
			SELECT *, ROW_NUMBER() OVER (PARTITION BY food_id ORDER BY date DESC) AS rn
			FROM daily_foods
		) AS df
		INNER JOIN foods f ON df.food_id = f.food_id
		WHERE df.rn = 1
		ORDER BY df.date DESC
		LIMIT $1
	`
	var entries []DailyFood
	if err := tx.Select(&entries, query, limit); err != nil {
		return entries, err
	}
	return entries, nil
}

// printFoodEntries prints food entries for a date.
func printFoodEntries(entries []DailyFood) {
	for i, entry := range entries {
		fmt.Printf("[%d] %s %s %.2f %s x %.2f serving\n", i+1, entry.Date.Format(dateFormat), entry.FoodName, entry.ServingSize, entry.ServingUnit, entry.NumberOfServings)
	}
}

// searchFoodLog uses date to search through logged foods.
func searchFoodLog(tx *sqlx.Tx, date time.Time) ([]DailyFood, error) {
	// Since DailyFood struct does not currently support time field, the
	// query excludes the time field from the selected records.
	const query = `
			SELECT df.id, df.food_id, df.meal_id, df.date, df.serving_size,
			df.number_of_servings, f.food_name, f.serving_unit
			FROM daily_foods df
			JOIN foods f ON df.food_id = f.food_id
			WHERE df.date = $1
	`
	var entries []DailyFood
	if err := tx.Select(&entries, query, date.Format(dateFormat)); err != nil {
		log.Printf("Search for food entries failed: %v\n", err)
		return nil, err
	}
	return entries, nil
}

// updateFoodEntry updates the given food entry in the database.
func updateFoodEntry(tx *sqlx.Tx, entryID int, f Food) error {
	const query = `
			UPDATE daily_foods
			SET serving_size = $1, number_of_servings = $2, calories = $3,
			protein = $4, fat = $5, carbs = $6, price = $7
			WHERE id = $8
	`
	_, err := tx.Exec(query, f.ServingSize, f.NumberOfServings, f.Calories,
		f.FoodMacros.Protein, f.FoodMacros.Fat, f.FoodMacros.Carbs, f.Price, entryID)
	if err != nil {
		log.Println("Failed to update entry in daily_foods.")
		return fmt.Errorf("updateFoodEntry: %w", err)
	}
	return nil
}

// DeleteFoodEntry deletes a logged food entry.
func DeleteFoodEntry(db *sqlx.DB) error {
	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
		return err
	}
	defer tx.Rollback()

	// Get selected weight entry.
	entry, err := selectFoodEntry(tx)
	if err != nil {
		return err
	}

	// Delete selected entry.
	if err := deleteOneFoodEntry(tx, entry.ID); err != nil {
		return err
	}

	fmt.Println("Successfully deleted food entry.")
	return tx.Commit()
}

// deleteOneFoodEntry deletes a logged food entry from the database.
func deleteOneFoodEntry(tx *sqlx.Tx, entryID int) error {
	const query = `
		DELETE FROM daily_foods
		WHERE id = $1
`
	if _, err := tx.Exec(query, entryID); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// ShowFoodLog fetches and prints entire food log.
func ShowFoodLog(db *sqlx.DB) error {
	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
		return err
	}
	defer tx.Rollback()

	entries, err := allFoodEntries(tx)
	if err != nil {
		return err
	}

	// Print food entries organized by date.
	var currentDate time.Time
	for _, entry := range entries {
		if !entry.Date.Equal(currentDate) {
			currentDate = entry.Date
			fmt.Printf("\n%v\n", currentDate.Format(("January 2, 2006")))
		}
		fmt.Printf("- %s: %.1f %s x %.1f serving | %.0f cals |\n",
			entry.FoodName, entry.ServingSize, entry.ServingUnit,
			entry.NumberOfServings, entry.Calories)
	}

	return tx.Commit()
}

// allFoodEntries retrieves all logged food entries. Ordered by most
// most recent date.
func allFoodEntries(tx *sqlx.Tx) ([]DailyFood, error) {
	// Since DailyFood struct does not currently support time field, the
	// queury excludes the time field from the selected records.
	const (
		query = `
			SELECT df.id, df.food_id, df.meal_id, df.date, df.serving_size,
			df.number_of_servings, df.calories, df.price, f.food_name,
			f.serving_unit
			FROM daily_foods df
			INNER JOIN foods f ON df.food_id = f.food_id
			ORDER BY df.date ASC
	`
		macrosQuery = `
	  	SELECT protein, fat, carbs
	  	FROM daily_foods
			WHERE id = $1
	`
	)

	var entries []DailyFood
	if err := tx.Select(&entries, query); err != nil {
		log.Println("Failed to get main details from daily food entries.")
		return nil, err
	}

	for i, entry := range entries {
		macros := &FoodMacros{}
		err := tx.Get(macros, macrosQuery, entry.ID)
		if err != nil {
			log.Printf("Failed to get macros from daily foods entries: %v\n", err)
			return nil, err
		}
		entries[i].FoodMacros = macros
	}

	return entries, nil
}

// LogMeal allows the user to create a new meal entry.
func LogMeal(db *sqlx.DB) error {
	tx, err := db.Beginx()
	defer tx.Rollback()
	if err != nil {
		return err
	}

	// Get selected meal.
	meal, err := selectMeal(db)
	if err != nil {
		return err
	}

	// Get the foods that make up the meal.
	mealFoods, err := MealFoodsWithPref(db, meal.ID)
	if err != nil {
		log.Println(err)
		return err
	}

	// If meal does not contain any foods, then return early
	if len(mealFoods) == 0 {
		log.Printf("Meal %q does not contain any foods.\n", meal.Name)
		return fmt.Errorf("Meal %q does not contain any foods.\n", meal.Name)
	}

	// Print the foods that make up the meal and their preferences.
	printMealDetails(mealFoods)

	// While user decides to change existing food preferences,
	for {
		// Get user response.
		response := promptUserEditDecision()

		// If user pressed <Enter>, break the loop.
		if response == "" {
			break
		}

		idx, err := strconv.Atoi(response)
		if err != nil {
			return fmt.Errorf("couldn't convert response to integer: %v", err)
		}

		// If user enters an invalid integer,
		if 1 > idx || idx > len(mealFoods) {
			fmt.Println("Number must be between 0 and number of foods. Please try again.")
			continue
		}

		// Get updated food preferences.
		f := promptMealFoodPref(mealFoods[idx-1].Food.ID, int64(meal.ID), mealFoods[idx-1].ServingSize, mealFoods[idx-1].NumberOfServings)

		// Make database update to meal food preferences.
		if err := UpdateMealFoodPrefs(tx, *f); err != nil {
			return err
		}
		fmt.Println("Updated food.")
	}

	// Get the updated foods that make up the meal.
	updatedMealFoods, err := MealFoodsWithPref(db, meal.ID)
	if err != nil {
		log.Println(err)
		return err
	}

	// Get date of meal entry.
	date := promptDateNotPast("Enter meal entry date")

	// Log selected meal to the meal log database table. Taking into
	// account food preferences.
	if err := AddMealEntry(tx, meal.ID, date); err != nil {
		log.Println(err)
		return err
	}

	// Bulk insert the foods that make up the meal into the daily_foods table.
	err = AddMealFoodEntries(tx, meal.ID, updatedMealFoods, date)
	if err != nil {
		log.Println(err)
		return err
	}

	fmt.Println("Successfully added meal entry.")

	return tx.Commit()
}

// selectMeal prints the user's meals, prompts them to select a meal,
// and returns the selected meal.
func selectMeal(db *sqlx.DB) (Meal, error) {
	// Get recently logged meals
	meals, err := MealsWithRecentFirst(db)
	if err != nil {
		return Meal{}, err
	}

	// Print all meals.
	for i, meal := range meals {
		fmt.Printf("[%d] %s\n", i+1, meal.Name)
	}

	// Get response.
	response := promptSelectResponse("meal")
	idx, err := strconv.Atoi(response)

	// While response is an integer
	for err == nil {
		// If integer is invalid,
		if 1 > idx || idx > len(meals) {
			fmt.Println("Number must be between 0 and number of meals. Please try again.")
			response = promptSelectResponse("meal")
			idx, err = strconv.Atoi(response)
			continue
		}
		// Otherwise, return food at valid index.
		return meals[idx-1], nil
	}
	// User response was a search term.

	// While user response is not an integer
	for {
		// Get the filtered meals.
		filteredMeals, err := SearchMeals(db, response)
		if err != nil {
			return Meal{}, err
		}

		// If no matches found,
		if len(filteredMeals) == 0 {
			fmt.Println("No matches found. Please try again.")
			response = promptSelectResponse("meal")
			continue
		}

		// Print meals.
		for i, meal := range filteredMeals {
			fmt.Printf("[%d] %s\n", i+1, meal.Name)
		}

		response = promptSelectResponse("meal")
		idx, err := strconv.Atoi(response)

		// While response is an integer
		for err == nil {
			// If integer is invalid,
			if 1 > idx || idx > len(filteredMeals) {
				fmt.Println("Number must be between 0 and number of meals. Please try again.")
				response = promptSelectResponse("meal")
				idx, err = strconv.Atoi(response)
				continue
			}
			// Otherwise, return food at valid index.
			return filteredMeals[idx-1], nil
		}
		// User response was a search term. Continue to next loop.
	}
}

// MealsWithRecentFirst retrieves the meals that have been logged
// recently first and then retrieves the remaining meals.
func MealsWithRecentFirst(db *sqlx.DB) ([]Meal, error) {
	const query = `
	SELECT meals.*
	FROM meals
	LEFT JOIN (
		SELECT meal_id, MAX(date) AS latest_date
		FROM daily_meals
		GROUP BY meal_id
	) AS dm
	ON meals.meal_id = dm.meal_id
	ORDER BY dm.latest_date DESC, meals.meal_id;
`

	var meals []Meal
	if err := db.Select(&meals, query); err != nil {
		return nil, err
	}

	for i, _ := range meals {
		m := &meals[i]
		mealFoods, err := MealFoodsWithPref(db, m.ID)
		if err != nil {
			return nil, err
		}
		m.Foods = mealFoods
		m.Cals = totalCals(mealFoods)
		m.Protein, m.Carbs, m.Fats = totalMacros(mealFoods)
	}

	return meals, nil
}

// totalCals returns the total calories for a given slice of meal foods.
func totalCals(foods []MealFood) float64 {
	var total float64
	for _, mf := range foods {
		total += mf.Calories
	}
	return total
}

// totalCals returns the total macros for a given slice of meal foods.
func totalMacros(foods []MealFood) (float64, float64, float64) {
	var protein, carbs, fats float64
	for _, mf := range foods {
		protein += mf.Food.FoodMacros.Protein
		carbs += mf.Food.FoodMacros.Carbs
		fats += mf.Food.FoodMacros.Fat
	}
	return protein, carbs, fats
}

// SearchMeals searches through meals slice and returns meals that
// contain the search term.
func SearchMeals(db *sqlx.DB, response string) ([]Meal, error) {
	var meals []Meal

	// Prioritize exact match, then match meals where `meal_name` starts
	// with the search term, and finally any meals where the `meal_name`
	// contains the search term.
	const query = `
			SELECT * FROM meals
			WHERE meal_name LIKE $1
			ORDER BY
					CASE
							WHEN meal_name = $2 THEN 1
							WHEN meal_name LIKE $3 THEN 2
							ELSE 3
					END
			LIMIT $4`

	// Search for meals in the database
	err := db.Select(&meals, query, "%"+response+"%", response, response+"%", SearchLimit)
	if err != nil {
		return nil, err
	}

	for i, _ := range meals {
		m := &meals[i]
		mealFoods, err := MealFoodsWithPref(db, m.ID)
		if err != nil {
			return nil, err
		}
		m.Foods = mealFoods
		m.Cals = totalCals(mealFoods)
		m.Protein, m.Carbs, m.Fats = totalMacros(mealFoods)
	}

	return meals, nil
}

// MealFoodsWithPref retrieves all the foods that make up a meal.
func MealFoodsWithPref(db *sqlx.DB, mealID int) ([]MealFood, error) {
	const query = `
	  SELECT food_id
		FROM meal_foods
		WHERE meal_id = $1
	`

	// First, get all the food IDs for the given meal.
	var foodIDs []int
	if err := db.Select(&foodIDs, query, mealID); err != nil {
		return nil, err
	}

	// Now, for each food ID, get the full food details and preferences.
	var mealFoods []MealFood
	for _, foodID := range foodIDs {
		mf, err := mealFoodWithPref(db, foodID, int64(mealID))
		if err != nil {
			return nil, err
		}
		mf.MealID = mealID
		mealFoods = append(mealFoods, mf)
	}

	return mealFoods, nil
}

// mealFoodWithPref retrieves one of the foods for a given meal,
// along its preferences.
func mealFoodWithPref(db *sqlx.DB, foodID int, mealID int64) (MealFood, error) {
	const (
		selectSQL = `
		SELECT * FROM foods
		WHERE food_id = $1"
`
		// Get the serving size and number of servings, preferring
		// meal_food_prefs and then food_prefs and then default
		servingSQL = `
			SELECT
					COALESCE(mfp.serving_size, fp.serving_size, f.serving_size, 100) AS serving_size,
					COALESCE(mfp.number_of_servings, fp.number_of_servings, 1) AS number_of_servings
			FROM foods f
			LEFT JOIN meal_food_prefs mfp ON mfp.food_id = f.food_id AND mfp.meal_id = $1
			LEFT JOIN food_prefs fp ON fp.food_id = f.food_id
			WHERE f.food_id = $2
			LIMIT 1
	`
		nutrientSQL = `
 	SELECT amount FROM food_nutrients WHERE food_id = ? AND nutrient_id IN (SELECT nutrient_id FROM nutrients WHERE nutrient_name = 'Energy' AND unit_name = 'KCAL' LIMIT 1)
	`
	)
	mf := MealFood{}

	// Get the food details
	if err := db.Get(&mf.Food, selectSQL, foodID); err != nil {
		log.Println("Failed to get food.")
		return MealFood{}, err
	}

	// Execute the SQL query and assign the result to the MealFood struct
	if err := db.Get(&mf, selectSQL, mealID, foodID); err != nil {
		log.Println("Failed to select serving size and number of servings.")
		return MealFood{}, err
	}

	// Execute the SQL query and assign the result to the calories field
	// in the MealFood struct
	if err := db.Get(&mf.Food.Calories, nutrientSQL, foodID); err != nil {
		log.Println("Failed to select portion calories.")
		return MealFood{}, err
	}

	// Get the macros for the food
	var err error
	mf.Food.FoodMacros, err = foodMacros(db, foodID)
	if err != nil {
		log.Println("Failed to get food macros.")
		return MealFood{}, err
	}

	ratio := mf.ServingSize / PortionSize
	mf.Food.Calories *= ratio * mf.NumberOfServings
	mf.Food.FoodMacros.Protein *= ratio * mf.NumberOfServings
	mf.Food.FoodMacros.Fat *= ratio * mf.NumberOfServings
	mf.Food.FoodMacros.Carbs *= ratio * mf.NumberOfServings
	mf.Food.Price *= ratio * mf.NumberOfServings

	return mf, nil
}

// FoodWithPref retrieves one food, along its preferences.
func FoodWithPref(db *sqlx.DB, foodID int) (*Food, error) {
	const (
		selectSQL = `
SELECT * FROM foods
WHERE food_id = $1
`
		// Override existing serving size and number of servings if there
		// exists a matching entry in the food_prefs table for the food id.
		servingSQL = `
        SELECT
            COALESCE(fp.serving_size, f.serving_size, 100) AS serving_size,
            COALESCE(fp.number_of_servings, 1) AS number_of_servings
        FROM foods f
        LEFT JOIN food_prefs fp ON fp.food_id = f.food_id
        WHERE f.food_id = $1
        LIMIT 1
    `
		calSQL = `
      SELECT amount FROM food_nutrients
			WHERE food_id = ? AND nutrient_id IN (
			  SELECT nutrient_id FROM nutrients
				WHERE nutrient_name = 'Energy' AND unit_name = 'KCAL'
				LIMIT 1
			)
		`
	)
	f := Food{}

	if err := db.Get(&f, selectSQL, foodID); err != nil {
		log.Println("Failed to get food.")
		return nil, err
	}

	if err := db.Get(&f, servingSQL, foodID); err != nil {
		log.Println("Failed to select serving size and number of servings.")
		return nil, err
	}

	// Execute the SQL query and assign the result to the calories field
	// in the Food struct
	if err := db.Get(&f.Calories, calSQL, foodID); err != nil {
		log.Println("Failed to select portion calories.")
		return nil, err
	}

	// Get the macros for the food.
	var err error
	f.FoodMacros, err = foodMacros(db, foodID)
	if err != nil {
		log.Println("Failed to get food macros.")
		return nil, err
	}

	ratio := f.ServingSize / PortionSize
	f.Calories *= ratio * f.NumberOfServings
	f.FoodMacros.Protein *= ratio * f.NumberOfServings
	f.FoodMacros.Fat *= ratio * f.NumberOfServings
	f.FoodMacros.Carbs *= ratio * f.NumberOfServings
	f.Price *= ratio * f.NumberOfServings

	return &f, nil
}

// printMealDetails prints the foods that make up the meal and their preferences.
func printMealDetails(mealFoods []MealFood) {
	var priceTotal float64
	for i, mf := range mealFoods {
		fmt.Printf("[%d] ", i+1)
		printMealFood(mf)
		priceTotal += mf.Food.Price
	}
	fmt.Printf("Total estimated cost of meal: $%.2f\n", priceTotal)
}

// printMealFood prints details of a given MealFood object.
func printMealFood(mealFood MealFood) {
	fmt.Printf("%s: %.2f %s x %.2f serving, %.2f cals ($%.2f)\n",
		mealFood.Food.Name, mealFood.ServingSize, mealFood.Food.ServingUnit,
		mealFood.NumberOfServings, mealFood.Food.Calories, mealFood.Food.Price)

	fmt.Printf("    Macros: | Protein: %-3.2fg | Carbs: %-3.2fg | Fat: %-3.2fg |\n", mealFood.Food.FoodMacros.Protein, mealFood.Food.FoodMacros.Carbs, mealFood.Food.FoodMacros.Fat)
}

// promptUserEditDecision prompts the user to select one of foods that
// make up a meal to edit or <enter> to use existing values.
func promptUserEditDecision() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter index of food to edit [Press <Enter> for existing values]: ")
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("promptUserEditDecision: %v\n", err)
	}
	// Remove the newline character at the end of the string
	response = strings.TrimSpace(response)
	return response
}

// AddMealEntry inserts a meal entry into the database.
func AddMealEntry(tx *sqlx.Tx, mealID int, date time.Time) error {
	const query = `
    INSERT INTO daily_meals (meal_id, date, time)
    VALUES ($1, $2, $3)
    `
	_, err := tx.Exec(query, mealID, date.Format(dateFormat), date.Format(dateFormatTime))
	if err != nil {
		return err
	}
	return nil
}

// AddMealFoodEntries bulk inserts foods that make up the meal into the database.
func AddMealFoodEntries(tx *sqlx.Tx, mealID int, mealFoods []MealFood, date time.Time) error {
	// Prepare a statement for bulk insert
	stmt, err := tx.Preparex("INSERT INTO daily_foods (food_id, meal_id, date, time, serving_size, number_of_servings, calories, protein, fat, carbs, price) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Iterate over each food and insert into the database
	for _, mf := range mealFoods {
		_, err = stmt.Exec(mf.Food.ID, mealID, date.Format(dateFormat), date.Format(dateFormatTime), mf.ServingSize, mf.NumberOfServings, mf.Food.Calories, mf.Food.FoodMacros.Protein, mf.Food.FoodMacros.Fat, mf.Food.FoodMacros.Carbs, mf.Food.Price)
		if err != nil {
			log.Println("Failed to execute bulk meal food insert.")
			return err
		}
	}

	return err
}

// FoodLogSummary fetches and prints a food log summary.
func FoodLogSummary(db *sqlx.DB) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get total amount of foods logged in the database.
	total, err := totalFoodsLogged(tx)
	if err != nil {
		log.Printf("Failed to get total amount of foods: %v\n", err)
		return err
	}
	fmt.Printf("\nTotal Foods Logged: %d\n", total)

	// Get most frequently consumed foods.
	foods, err := frequentFoods(tx, 10)
	if err != nil {
		log.Printf("Failed to get frequent foods: %v\n", err)
		return err
	}

	fmt.Println("\nMost Frequently Consumed Foods:")

	// Print most frequent consumed foods.
	for _, food := range foods {
		fmt.Printf("- %s: eaten %d times\n", food.FoodName, food.Count)
	}

	return tx.Commit()
}

// totalFoodsLogged fetches and returns the total amount of foods
// entries logged in the database.
func totalFoodsLogged(tx *sqlx.Tx) (int, error) {
	const query = `
      SELECT COUNT(*)
      FROM daily_foods
    `
	var total int
	if err := tx.Get(&total, query); err != nil {
		return 0, err
	}
	return total, nil
}

// frequentFoods retrieves most recently logged food entries.
func frequentFoods(tx *sqlx.Tx, limit int) ([]DailyFoodCount, error) {
	// Since DailyFood struct does not currently support time field, the
	// query excludes the time field from the selected records.
	const query = `
    SELECT df.food_id, df.date, df.serving_size, df.number_of_servings,
		  f.food_name, f.serving_unit, COUNT(*) as count
    FROM daily_foods df
    INNER JOIN foods f ON df.food_id = f.food_id
    GROUP BY df.food_id, f.food_name, f.serving_unit
    ORDER BY count DESC
    LIMIT $1
  `
	var foods []DailyFoodCount
	if err := tx.Select(&foods, query, limit); err != nil {
		return nil, err
	}
	return foods, nil
}

// FoodLogSummaryDay prints the current nutritional totals for a given
// day and provides insight on progress towards nutritional goals.
func FoodLogSummaryDay(db *sqlx.DB, u *UserInfo) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get the food entries for the present day.
	entries, err := foodEntriesForDate(tx, time.Now())
	if err != nil {
		return err
	}

	// If there are zero entries for today, then return early.
	if len(entries) == 0 {
		fmt.Println("No foods logged for today.")
		return nil
	}

	var calorieTotal float64
	var proteinTotal float64
	var fatTotal float64
	var carbTotal float64
	var priceTotal float64

	// Calculate nutritional totals.
	for _, entry := range entries {
		calorieTotal += entry.Calories
		proteinTotal += entry.FoodMacros.Protein
		fatTotal += entry.FoodMacros.Fat
		carbTotal += entry.FoodMacros.Carbs
		priceTotal += entry.Price
	}

	// Get nutritional goals.
	calorieGoal := u.Phase.GoalCalories
	if u.Phase.Status != "active" {
		calorieGoal = u.TDEE
	}
	proteinGoal := u.Macros.Protein
	fatGoal := u.Macros.Fats
	carbGoal := u.Macros.Carbs

	printNutrientProgress(proteinTotal, proteinGoal, "Protein")
	printNutrientProgress(fatTotal, fatGoal, "Fat")
	printNutrientProgress(carbTotal, carbGoal, "Carbs")
	printCalorieProgress(calorieTotal, calorieGoal, "Calories")
	fmt.Printf("\n%.2f calories remaining.\n", calorieGoal-calorieTotal)
	fmt.Printf("Eaten $%.2f worth of food today.\n", priceTotal)

	return tx.Commit()
}

// foodEntriesForDate retrieves the food entries for a given date.
func foodEntriesForDate(tx *sqlx.Tx, date time.Time) ([]DailyFood, error) {
	const (
		// Since DailyFood struct does not currently support time field, the
		// queury excludes the time field from the selected records.
		query = `
      SELECT df.id, df.food_id, df.meal_id, df.date, df.serving_size,
	      df.number_of_servings, df.calories, df.price, f.food_name, f.serving_unit
      FROM daily_foods df
      INNER JOIN foods f ON df.food_id = f.food_id
	    WHERE date = $1
      ORDER BY df.date DESC
    `
		macrosQuery = `
      SELECT protein, fat, carbs
      FROM daily_foods
      WHERE id = $1
	  `
	)

	var entries []DailyFood
	if err := tx.Select(&entries, query, date.Format(dateFormat)); err != nil {
		log.Printf("Failed to get main details from daily food entries: %v\n", err)
		return nil, err
	}

	// Retrieve the macros.
	for i, entry := range entries {
		macros := &FoodMacros{}
		if err := tx.Get(macros, macrosQuery, entry.ID); err != nil {
			log.Printf("Failed to get macros from daily foods entries: %v\n", err)
			return nil, err
		}
		entries[i].FoodMacros = macros
	}

	return entries, nil
}

// printNutrientProgress prints the nutrient progress.
func printNutrientProgress(current, goal float64, name string) {
	progressBar := renderProgressBar(current, goal)
	fmt.Printf("%-9s %s %3.0f%% (%.0fg / %.0fg)\n", name+":", progressBar,
		current*100/goal, current, goal)
}

// printCalorieProgress prints the calories progress.
func printCalorieProgress(current, goal float64, name string) {
	progressBar := renderProgressBar(current, goal)
	fmt.Printf("%-9s %s %3.0f%% (%.0f / %.0f)\n", name+":", progressBar,
		current*100/goal, current, goal)
}

// renderProgressBar renders an ASCII progress bar.
func renderProgressBar(current, goal float64) string {
	const barLength = 10
	percentage := current / goal
	filledLength := int(percentage * float64(barLength))

	// If percentage is greater than 100%, set a full bar.
	if percentage > 100 {
		filledLength = barLength
	}

	bar := "["
	for i := 0; i < filledLength; i++ {
		bar += fullBlock
	}
	for i := filledLength; i < barLength; i++ {
		bar += lightBlock
	}
	bar += "]"
	return bar
}

// ValidLog creates and returns array containing the
// valid log entries.
//
// Assumptions:
// * Diet phase activity has been checked. That is, this function should
// not be called for a diet phase that is not currently active.
func ValidLog(u *UserInfo, entries *[]Entry) *[]Entry {
	today := time.Now()

	var subset []Entry
	for _, entry := range *entries {
		// Only consider dates that fall somewhere inbetween the diet
		// start date and the current date.
		if (entry.Date.After(u.Phase.StartDate) || isSameDay(entry.Date, u.Phase.StartDate)) && (entry.Date.Before(today) || isSameDay(entry.Date, today)) {
			subset = append(subset, entry)
		}
	}

	return &subset
}
