package calories

import (
	"bufio"
	"database/sql"
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

const (
	weightSearchLimit = 10
	dateFormatTime    = "15:04:05"
)

var ErrDone = errors.New("done")

// Entry fields will be constructed from daily_weights and daily_foods
// table during runtime.
// Nutrients are for portion size (100 serving unit)
type Entry struct {
	UserWeight float64   `db:"user_weight"`
	UserCals   float64   `db:"user_cals"`
	Date       time.Time `db:"date"`
	Protein    float64   `db:"protein"`
	Carbs      float64   `db:"carbs"`
	Fat        float64   `db:"fat"`
}

type WeightEntry struct {
	ID     int       `db:"id"`
	Date   time.Time `db:"date"`
	Weight float64   `db:"weight"`
}

type DailyFood struct {
	ID               int       `db:"id"`
	FoodID           int       `db:"food_id"`
	MealID           *int      `db:"meal_id"`
	Date             time.Time `db:"date"`
	ServingSize      float64   `db:"serving_size"`
	ServingUnit      string    `db:"serving_unit"`
	NumberOfServings float64   `db:"number_of_servings"`
	FoodName         string    `db:"food_name"`
}

type DailyFoodCount struct {
	DailyFood
	Count int `db:"count"`
}

// GetAllEntries returns all the user's entries from the database.
func GetAllEntries(db *sqlx.DB) (*[]Entry, error) {
	query := `
  SELECT
    dw.date,
    dw.weight AS user_weight,

    -- Calculate sum of calories and macros for each day, taking into account the serving size and the number of servings.
    -- If a nutrient is not logged for a particular day, its amount is treated as 0.
    -- If no preference is set for a food, default serving size is assumed to be 1 (to maintain the existing nutrient portion size).
    -- If a food is part of a meal, preference is taken from 'meal_food_prefs', otherwise from 'food_prefs'.
    SUM(
      CASE WHEN fn.nutrient_id = 1008
        THEN fn.amount * COALESCE(mfp.serving_size, fp.serving_size, 1)
                      * COALESCE(mfp.number_of_servings, fp.number_of_servings, 1)
        ELSE 0 END
    ) AS user_cals,

    SUM(
      CASE WHEN fn.nutrient_id = 1003
        THEN fn.amount * COALESCE(mfp.serving_size, fp.serving_size, 1)
                      * COALESCE(mfp.number_of_servings, fp.number_of_servings, 1)
        ELSE 0 END
    ) AS protein,

    SUM(
      CASE WHEN fn.nutrient_id = 1005
        THEN fn.amount * COALESCE(mfp.serving_size, fp.serving_size, 1)
                      * COALESCE(mfp.number_of_servings, fp.number_of_servings, 1)
        ELSE 0 END
    ) AS carbs,

    SUM(
      CASE WHEN fn.nutrient_id = 1004
        THEN fn.amount * COALESCE(mfp.serving_size, fp.serving_size, 1)
                      * COALESCE(mfp.number_of_servings, fp.number_of_servings, 1)
        ELSE 0 END
    ) AS fat

  FROM daily_weights dw -- User's weight data.
		-- Join daily food data on date. Only if food_id is not null.
    JOIN daily_foods df ON dw.date = df.date AND df.food_id IS NOT NULL
		-- Join with food nutrients data on food_id.
    JOIN food_nutrients fn ON df.food_id = fn.food_id
		-- Join with food preferences data on food_id. This data is used when food is consumed outside of a meal.
    LEFT JOIN food_prefs fp ON df.food_id = fp.food_id
		-- Join with meal food preferences data on food_id and meal_id. This data is used when food is consumed as part of a meal.
    LEFT JOIN meal_food_prefs mfp ON df.food_id = mfp.food_id AND df.meal_id = mfp.meal_id

	-- Filter only specific nutrient_ids.
  WHERE fn.nutrient_id IN (1008, 1003, 1005, 1004)

	-- Group by date and user weight to aggregate nutrition data by day.
  GROUP BY dw.date, dw.weight

	-- Ensure groups include at least one food_id, which indicates at least one food was logged for that day.
  HAVING SUM(df.food_id) IS NOT NULL

	-- Sort results by date.
  ORDER BY dw.date
`

	var entries []Entry
	err := db.Select(&entries, query)
	if err != nil {
		log.Fatalf("GetAllEntries: %v\n", err)
	}

	return &entries, nil
}

/*
// ReadEntries reads user entries from CSV file into a dataframe.
func ReadEntries() (*dataframe.DataFrame, error) {
	// Does entries file exist?
	if _, err := os.Stat(EntriesFilePath); os.IsNotExist(err) {
		log.Println("ERROR: Entries file not found.")
		return nil, err
	}

	// Open entries file
	csvfile, err := os.Open(EntriesFilePath)
	if err != nil {
		log.Printf("ERROR: Couldn't open %s\n", EntriesFilePath)
		return nil, err
	}
	defer csvfile.Close()

	// Read entries from CSV into a dataframe.
	ctx := context.TODO()
	logs, err := imports.LoadFromCSV(ctx, csvfile)
	if err != nil {
		log.Printf("ERROR: Couldn't read %s\n", EntriesFilePath)
		return nil, err
	}

	return logs, nil
}
*/

// LogWeight gets weight and date from user to create a new weight entry.
func LogWeight(u *UserInfo, db *sqlx.DB) {
	for {
		date := getDateNotPast("Enter weight entry date")
		weight, err := getWeight(u.System)
		if err != nil {
			fmt.Printf("%v. Please try again.\n", err)
			continue
		}
		err = addWeightEntry(db, date, weight)
		if err != nil {
			fmt.Printf("%v. Please try again.\n", err)
			continue
		}
		break
	}
}

// addWeightEntry inserts a weight entry into the database.
func addWeightEntry(db *sqlx.DB, date time.Time, weight float64) error {
	// Ensure weight hasn't already been logged for given date.
	exists, err := checkWeightExists(db, date)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("Weight for this date has already been logged.")
	}

	// Insert the new weight entry into the weight database.
	_, err = db.Exec(`INSERT INTO daily_weights (date, time, weight) VALUES ($1, $2, $3)`, date.Format(dateFormat), date.Format(dateFormatTime), weight)
	if err != nil {
		return err
	}

	fmt.Println("Added weight entry.")
	return nil
}

// getDateNotPast prompts user for date that it not in the past, validates user
// response until user enters a valid date, and return the valid date.
func getDateNotPast(s string) (date time.Time) {
	for {
		// Prompt user for diet start date.
		r := promptDate(fmt.Sprintf("%s (YYYY-MM-DD) [Press Enter for today's date]: ", s))

		// If user entered default date,
		if r == "" {
			// set date to today's date.
			r = time.Now().Format(dateFormat)
		}

		// Ensure user response is a date.
		var err error
		date, err = validateDateStr(r)
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
	log, err := getAllWeightEntries(db)
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

	// Update entry.
	err = updateWeightEntry(db, entry.ID, weight)
	if err != nil {
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
	// Start a new transaction
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	// Execute the update statement
	_, err = tx.Exec(`
			UPDATE daily_weights
			SET weight = $1
			WHERE id = $2
			`, newWeight, id)

	// If there was an error executing the query, return the error
	if err != nil {
		return err
	}

	// If everything went fine, commit the transaction
	return tx.Commit()
}

// DeleteWeightEntry deletes a weight entry.
func DeleteWeightEntry(db *sqlx.DB) error {
	// Get selected weight entry.
	entry, err := selectWeightEntry(db)
	if err != nil {
		return err
	}

	// Delete selected entry.
	err = deleteOneWeightEntry(db, entry.ID)
	if err != nil {
		return err
	}
	fmt.Println("Deleted weight entry.")

	return nil
}

// deleteOneWeightEntry deletes one weight entry from the database.
func deleteOneWeightEntry(db *sqlx.DB, id int) error {
	// Start a new transaction
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	// Execute the delete statement
	_, err = tx.Exec(`
      DELETE FROM daily_weights
      WHERE id = $1
      `, id)

	// If there was an error executing the query, return the error
	if err != nil {
		return err
	}

	// If everything went fine, commit the transaction
	return tx.Commit()
}

// selectWeightEntry prints the user's weight entries, prompts them to select
// a weight entry, and returns the selected weight entry.
func selectWeightEntry(db *sqlx.DB) (WeightEntry, error) {
	// Get all weight logs.
	log, err := getRecentWeightEntries(db)
	if err != nil {
		return WeightEntry{}, err
	}

	// Print recent weight entries.
	printWeightEntries(log)

	// Get response.
	response := promptSelectEntry()
	idx, err := strconv.Atoi(response)

	// While response is an integer
	for err == nil {
		// If integer is invalid,
		if 1 > idx || idx > len(log) {
			fmt.Println("Number must be between 0 and number of entries. Please try again.")
			response = promptSelectEntry()
			idx, err = strconv.Atoi(response)
			continue
		}
		// Otherwise, return food at valid index.
		return log[idx-1], nil
	}
	// User response was a date to search.

	// While user response is not an integer,
	for {
		// Validate user response.
		date, err := validateDateStr(response)
		if err != nil {
			fmt.Printf("%v. Please try again.", err)
			response = promptSelectEntry()
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
			response = promptSelectEntry()
			continue
		}

		// Print entry.
		fmt.Printf("[1] %s %f\n", entry.Date.Format(dateFormat), entry.Weight)

		response = promptSelectEntry()
		idx, err := strconv.Atoi(response)

		// While response is an integer
		for err == nil {
			// If integer is invalid,
			if idx != 1 {
				fmt.Println("Number must be 1. Please try again.")
				response = promptSelectEntry()
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

// getAllWeightEntries returns all the user's logged weight entries.
func getAllWeightEntries(db *sqlx.DB) ([]WeightEntry, error) {
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

// getRecentWeightEntries returns the user's logged weight entries up to
// a limit.
func getRecentWeightEntries(db *sqlx.DB) ([]WeightEntry, error) {
	// Since DailyWeight struct does not currently support time field, the
	// queury excludes the time field from the selected records.
	const query = `
		SELECT id, date, weight FROM daily_weights ORDER by date DESC LIMIT $1
		`

	wl := []WeightEntry{}
	err := db.Select(&wl, query, weightSearchLimit)
	if err != nil {
		return nil, err
	}
	return wl, nil
}

// promptSelectEntry prompts and returns entry to select or a search
// term.
func promptSelectEntry() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter entry index to select or date to search (YYYY-MM-DD): ")
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	// Remove the newline character at the end of the string
	response = strings.TrimSpace(response)
	return response
}

// searchWeightLog searchs through all weight entries and returns the
// entry that matches the entered date.
func searchWeightLog(db *sqlx.DB, d time.Time) (*WeightEntry, error) {
	// Since DailyWeight struct does not currently support time field, the
	// queury excludes the time field from the selected records.
	const query = `
		SELECT id, date, weight FROM daily_weights ORDER by date = $1 LIMIT 1
		`

	var entry WeightEntry
	// Search for weight entry in the database
	err := db.Get(&entry, query, d.Format(dateFormat))
	if err != nil {
		log.Printf("Search for weight entry failed: %v\n", err)
		return nil, err
	}

	return &entry, nil
}

// checkWeightExists checks if a weight entry already exists for the
// given date.
func checkWeightExists(db *sqlx.DB, date time.Time) (bool, error) {
	var count int
	err := db.Get(&count, `SELECT COUNT(*) FROM daily_weights WHERE date = $1`, date.Format(dateFormat))
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// LogFood gets selected food user to create a new food entry.
func LogFood(db *sqlx.DB) error {
	// Start a new transaction
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	// TODO: Display most recently selected foods
	// 			 Refactor selectFood to be able to pick from one of the selected
	// 			 foods or search term.
	// Get selected food
	food, err := selectFood(tx)
	if err != nil {
		if errors.Is(err, ErrDone) {
			fmt.Println("No food selected.")
			return nil // Not really an "error" situation
		}
		return err
	}

	// Get any existing preferences for the selected food.
	f, err := getFoodPref(tx, food.ID)
	if err != nil {
		log.Println(err)
		return err
	}

	// Display any existing preferences for the selected food.
	printFoodPref(*f)

	var s string
	fmt.Printf("Do you want to update these values? (y/n): ")
	fmt.Scan(&s)

	// If the user decides to change existing food preferences,
	if strings.ToLower(s) == "y" {
		// Get updated food preferences.
		f = getFoodPrefUserInput(food.ID)
		// Make database update for food preferences.
		err := updateFoodPrefs(tx, f)
		if err != nil {
			return err
		}
	}

	// Get date of food entry.
	date := getDateNotPast("Enter food entry date")

	// Log selected food to the food log database table. Taking into
	// account food preferences.
	err = addFoodEntry(tx, f, date)
	if err != nil {
		log.Println(err)
		return err
	}
	fmt.Println("Added food entry.")

	return tx.Commit()
}

// selectFood prompts the user to enter a search term, prints the matched
// foods, prompts user to enter an index to select a food or another
// serach term for a different food. This repeats until user enters a
// valid index.
func selectFood(tx *sqlx.Tx) (Food, error) {
	// Get initial search term.
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter food name or 'done': ")
	response, err := reader.ReadString('\n')
	if err != nil {
		return Food{}, fmt.Errorf("Failed to read string. %v", err)
	}
	// Remove the newline character at the end of the string.
	response = strings.TrimSpace(response)

	// If user enters "done", then return early.
	if response == "done" {
		return Food{}, ErrDone
	}

	// While user response is not an integer
	for {
		// Get filtered foods.
		filteredFoods, err := searchFoods(tx, response)
		if err != nil {
			return Food{}, err
		}

		// If no matches found,
		if len(*filteredFoods) == 0 {
			fmt.Println("No matches found. Please try again.")
			response = promptSelectResponse("food")
			continue
		}

		// Print foods.
		for i, food := range *filteredFoods {
			fmt.Printf("[%d] %s\n", i+1, food.Name)
		}

		response = promptSelectResponse("food")
		idx, err := strconv.Atoi(response)

		// While response is an integer
		for err == nil {
			// If integer is invalid,
			if 1 > idx || idx > len(*filteredFoods) {
				fmt.Println("Number must be between 0 and number of foods. Please try again.")
				response = promptSelectResponse("food")
				idx, err = strconv.Atoi(response)
				continue
			}
			// Otherwise, return food at valid index.
			return (*filteredFoods)[idx-1], nil
		}
		// User response was a search term. Continue to next loop.
	}
}

// searchFoods searchs through all foods and returns food that contain
// the search term.
func searchFoods(tx *sqlx.Tx, response string) (*[]Food, error) {
	var foods []Food

	// Prioritize exact match, then match foods where `food_name` starts
	// with the search term, and finally any foods where the `food_name`
	// contains the search term.
	query := `
        SELECT * FROM foods
        WHERE food_name LIKE $1
        ORDER BY
            CASE
                WHEN food_name = $2 THEN 1
                WHEN food_name LIKE $3 THEN 2
                ELSE 3
            END
        LIMIT $4`

	// Search for foods in the database
	err := tx.Select(&foods, query, "%"+response+"%", response, response+"%", searchLimit)
	if err != nil {
		log.Printf("Search for foods failed: %v\n", err)
		return nil, err
	}

	return &foods, nil
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
	query := `
    SELECT
      f.food_id,
      COALESCE(fp.serving_size, f.serving_size) AS serving_size,
			COALESCE(fp.number_of_servings, 1) AS number_of_servings,
			f.ServingUnit
    FROM foods f
    LEFT JOIN food_prefs fp ON f.food_id = fp.food_id
    WHERE f.food_id = ?
  `

	var pref FoodPref
	err := tx.Get(&pref, query, foodID)

	if err != nil {
		// Handle a case when no preference found
		if err == sql.ErrNoRows {
			// If no rows are found, return an empty FoodPref struct with a custom error
			return &FoodPref{}, fmt.Errorf("no preference found for food ID %d", foodID)
		}
		return &FoodPref{}, fmt.Errorf("unable to execute query: %w", err)
	}

	return &pref, nil
}

// printFoodPref prints the perferences for a food.
func printFoodPref(pref FoodPref) {
	// TODO: add unit and household serving size to serving size
	fmt.Printf("Serving size: %.2f %s", math.Round(100*pref.ServingSize)/100, pref.ServingUnit)
	fmt.Printf("Number of serving: %.1f\n", math.Round(10*pref.NumberOfServings)/10)
}

// getFoodPrefUserInput prompts user for food perferences, validates their
// response until they've entered a valid response, and returns the
// valid response.
func getFoodPrefUserInput(foodID int) *FoodPref {
	pref := &FoodPref{}

	pref.FoodID = foodID
	pref.ServingSize, pref.NumberOfServings = getServingSizeAndNumServings()

	return pref
}

// getMealFoodPrefUserInput prompts user for meal food perferences,
// validates their response until they've entered a valid response,
// and returns the valid response.
func getMealFoodPrefUserInput(foodID int, mealID int64) *MealFoodPref {
	pref := &MealFoodPref{}

	pref.FoodID = foodID
	pref.MealID = mealID
	pref.ServingSize, pref.NumberOfServings = getServingSizeAndNumServings()

	return pref
}

// getServingSizeAndNumServings prompts user for serving size and number
// of servings, validates their response until they've entered a valid
// response, and returns the valid response.
func getServingSizeAndNumServings() (float64, float64) {
	return getServingSize(), getNumServings()
}

// updateFoodPrefs updates the user's preferences for a given
// food.
func updateFoodPrefs(tx *sqlx.Tx, pref *FoodPref) error {
	// Execute the update statement
	_, err := tx.NamedExec(`
			INSERT INTO food_prefs (food_id, number_of_servings, serving_size)
      VALUES (:food_id, :number_of_servings, :serving_size)
      ON CONFLICT(food_id) DO UPDATE SET
      number_of_servings = :number_of_servings,
      serving_size = :serving_size`,
		pref)

	// If there was an error executing the query, return the error
	if err != nil {
		log.Printf("Failed to update food prefs: %v\n", err)
		return err
	}

	return nil
}

// addFoodEntry inserts a food entry into the database.
func addFoodEntry(tx *sqlx.Tx, pref *FoodPref, date time.Time) error {
	const query = `
		INSERT INTO daily_foods	(food_id, date, time, serving_size, number_of_servings)
		VALUES ($1, $2, $3, $4, $5)
		`

	_, err := tx.Exec(query, pref.FoodID, date.Format(dateFormat), date.Format(dateFormatTime), pref.ServingSize, pref.NumberOfServings)
	// If there was an error executing the query, return the error
	if err != nil {
		return err
	}

	return nil
}

// UpdateFoodLog updates an existing food entry in the database.
func UpdateFoodLog(db *sqlx.DB) error {
	// Start a new transaction
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	// Let user select food entry to update.
	entry, err := selectFoodEntry(tx)
	if err != nil {
		return err
	}

	// Get new food preferences.
	pref := getFoodPrefUserInput(entry.FoodID)

	// Update food entry.
	err = updateFoodEntry(tx, entry.ID, *pref)
	if err != nil {
		return err
	}
	fmt.Println("Updated food entry.")

	return tx.Commit()
}

// selectFoodEntry prints recently logged foods, prompts user to enter a
// search term, prompts user to enter an index to select a food entry or
// another serach term for a different food entry. This repeats until
// user enters a valid index.
func selectFoodEntry(tx *sqlx.Tx) (DailyFood, error) {
	// Get most recently logged foods.
	recentFoods, err := getRecentFoodEntries(tx, searchLimit)
	if err != nil {
		log.Println(err)
		return DailyFood{}, err
	}

	// Print recent food entries.
	printFoodEntries(recentFoods)

	// Get response.
	response := promptSelectEntry()
	idx, err := strconv.Atoi(response)

	// While response is an integer
	for err == nil {
		// If integer is invalid,
		if 1 > idx || idx > len(recentFoods) {
			fmt.Println("Number must be between 0 and number of entries. Please try again.")
			response = promptSelectEntry()
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
		date, err := validateDateStr(response)
		if err != nil {
			fmt.Printf("%v. Please try again.", err)
			response = promptSelectEntry()
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
			response = promptSelectEntry()
			continue
		}

		// Print the foods entries for given date.
		printFoodEntries(filteredEntries)

		response = promptSelectEntry()
		idx, err := strconv.Atoi(response)

		// While response is an integer
		for err == nil {
			// If integer is invalid,
			if idx != 1 {
				fmt.Println("Number must be between 0 and number of entries. Please try again.")
				response = promptSelectEntry()
				idx, err = strconv.Atoi(response)
				continue
			}
			// Otherwise, return entry at valid index.
			return filteredEntries[idx-1], nil
		}
		// User response was a search term. Continue to next loop.
	}
}

// getRecentFoodEntries retrieves most recently logged food entries.
func getRecentFoodEntries(tx *sqlx.Tx, limit int) ([]DailyFood, error) {
	// Since DailyFood struct does not currently support time field, the
	// query excludes the time field from the selected records.
	const query = `
        SELECT df.id, df.food_id, df.meal_id, df.date, df.serving_size, df.number_of_servings, f.food_name, f.serving_unit
        FROM daily_foods df
        INNER JOIN foods f ON df.food_id = f.food_id
        ORDER BY df.date DESC
        LIMIT $1
    `

	var entries []DailyFood
	if err := tx.Select(&entries, query, limit); err != nil {
		return nil, err
	}

	return entries, nil
}

// printFoodEntries prints food entries for a date.
func printFoodEntries(entries []DailyFood) {
	for i, entry := range entries {
		fmt.Printf("[%d] %s %s \n", i+1, entry.Date.Format(dateFormat), entry.FoodName)
		fmt.Printf("Serving size: %.2f %s\n", math.Round(100*entry.ServingSize)/100, entry.ServingUnit)
		fmt.Printf("Number of servings: %.1f\n", math.Round(10*entry.NumberOfServings)/10)
	}
}

// searchFoodLog uses date to search through logged foods.
func searchFoodLog(tx *sqlx.Tx, date time.Time) ([]DailyFood, error) {
	// Since DailyFood struct does not currently support time field, the
	// query excludes the time field from the selected records.
	const query = `
        SELECT df.id, df.food_id, df.meal_id, df.date, df.serving_size, df.number_of_servings, f.food_name, f.serving_unit
    		FROM daily_foods df
    		JOIN foods f ON df.food_id = f.food_id
    		WHERE df.date = $1
  	`

	var entries []DailyFood
	// Search for food entries in the database for given date.
	err := tx.Select(&entries, query, date.Format(dateFormat))
	if err != nil {
		log.Printf("Search for food entries failed: %v\n", err)
		return nil, err
	}

	return entries, nil
}

// updateFoodEntry updates the given food entry in the database.
func updateFoodEntry(tx *sqlx.Tx, entryID int, pref FoodPref) error {
	query := `
        UPDATE daily_foods
        SET serving_size = $1, number_of_servings = $2
        WHERE id = $3
    `

	// Execute the update statement
	_, err := tx.Exec(query, pref.ServingSize, pref.NumberOfServings, entryID)

	// If there was an error executing the query, return the error
	if err != nil {
		return fmt.Errorf("updateFoodEntry: %w", err)
	}

	return nil
}

// DeleteFoodEntry deletes a logged food entry.
func DeleteFoodEntry(db *sqlx.DB) error {
	// Start a new transaction
	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
		return err
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	// Get selected weight entry.
	entry, err := selectFoodEntry(tx)
	if err != nil {
		return err
	}

	// Delete selected entry.
	err = deleteOneFoodEntry(tx, entry.ID)
	if err != nil {
		return err
	}
	fmt.Println("Deleted weight entry.")

	return tx.Commit()
}

// deleteOneFoodEntry deletes a logged food entry from the database.
func deleteOneFoodEntry(tx *sqlx.Tx, entryID int) error {
	// Execute the delete statement
	_, err := tx.Exec(`
      DELETE FROM daily_foods
      WHERE id = $1
      `, entryID)

	// If there was an error executing the query, return the error
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// ShowFoodLog fetches and prints entire food log.
func ShowFoodLog(db *sqlx.DB) error {
	// Start a new transaction
	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
		return err
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	entries, err := getAllFoodEntries(tx)
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
		fmt.Printf("- %s, Serving size: %.2f %s Num Servings: %.2f\n", entry.FoodName, entry.ServingSize, entry.ServingUnit, entry.NumberOfServings)
	}

	return tx.Commit()
}

// getAllFoodEntries retrieves all logged food entries. Ordered by most
// most recent date.
func getAllFoodEntries(tx *sqlx.Tx) ([]DailyFood, error) {
	// Since DailyFood struct does not currently support time field, the
	// queury excludes the time field from the selected records.
	const query = `
        SELECT df.id, df.food_id, df.meal_id, df.date, df.serving_size, df.number_of_servings, f.food_name, f.serving_unit
        FROM daily_foods df
        INNER JOIN foods f ON df.food_id = f.food_id
        ORDER BY df.date DESC
    `
	var entries []DailyFood
	if err := tx.Select(&entries, query); err != nil {
		return nil, err
	}

	return entries, nil
}

// LogMeal allows the user to create a new meal entry.
func LogMeal(db *sqlx.DB) error {
	// Start a new transaction
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	// Get selected meal.
	meal, err := selectMeal(tx)
	if err != nil {
		return err
	}

	// Get the foods that make up the meal.
	mealFoods, err := getMealFoodsWithPref(tx, meal.ID)
	if err != nil {
		log.Println(err)
		return err
	}

	// Print the foods that make up the meal and their preferences.
	printMealDetails(mealFoods)

	// While user decides to change existing food preferences,
	for {
		// Get user response.
		response := promptUserEditDecision()

		// If the user pressed <enter>, break the loop.
		if response == "" {
			break
		}

		idx, err := strconv.Atoi(response)

		// If user enters an invalid integer,
		if 1 > idx || idx > len(mealFoods) {
			fmt.Println("Number must be between 0 and number of foods. Please try again.")
			continue
		}

		// Get updated food preferences.
		f := getMealFoodPrefUserInput(mealFoods[idx-1].Food.ID, int64(meal.ID))

		// Make database update to meal food preferences.
		err = updateMealFoodPrefs(tx, f)
		if err != nil {
			return err
		}
		fmt.Println("Updated food.")
	}

	// Get date of meal entry.
	date := getDateNotPast("Enter meal entry date")

	// Log selected meal to the meal log database table. Taking into
	// account food preferences.
	err = addMealEntry(tx, meal, date)
	if err != nil {
		log.Println(err)
		return err
	}

	// Bulk insert the foods that make up the meal into the daily_foods table.
	err = addMealFoodEntries(tx, meal.ID, mealFoods, date)
	if err != nil {
		log.Println(err)
		return err
	}

	fmt.Println("Added meal entry.")

	return tx.Commit()
}

// selectMeal prints the user's meals, prompts them to select a meal,
// and returns the selected meal.
func selectMeal(tx *sqlx.Tx) (Meal, error) {
	// Get all meals.
	meals, err := getAllMeals(tx)
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

	// While user reponse is not an integer
	for {
		// Get the filtered meals.
		filteredMeals, err := searchMeals(tx, response)
		if err != nil {
			return Meal{}, err
		}

		// If no matches found,
		if len(*filteredMeals) == 0 {
			fmt.Println("No matches found. Please try again.")
			response = promptSelectResponse("meal")
			continue
		}

		// Print meals.
		for i, meal := range *filteredMeals {
			fmt.Printf("[%d] %s\n", i+1, meal.Name)
		}

		response = promptSelectResponse("meal")
		idx, err := strconv.Atoi(response)

		// While response is an integer
		for err == nil {
			// If integer is invalid,
			if 1 > idx || idx > len(*filteredMeals) {
				fmt.Println("Number must be between 0 and number of meals. Please try again.")
				response = promptSelectResponse("meal")
				idx, err = strconv.Atoi(response)
				continue
			}
			// Otherwise, return food at valid index.
			return (*filteredMeals)[idx-1], nil
		}
		// User response was a search term. Continue to next loop.
	}
}

// searchMeals searches through meals slice and returns meals that
// contain the search term.
func searchMeals(tx *sqlx.Tx, response string) (*[]Meal, error) {
	var meals []Meal

	// Prioritize exact match, then match meals where `meal_name` starts
	// with the search term, and finally any meals where the `meal_name`
	// contains the search term.
	query := `
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
	err := tx.Select(&meals, query, "%"+response+"%", response, response+"%", searchLimit)
	if err != nil {
		log.Printf("Search for meals failed: %v\n", err)
		return nil, err
	}

	return &meals, nil
}

// getMealFoodsWithPref retrieves all the foods that make up a meal.
func getMealFoodsWithPref(tx *sqlx.Tx, mealID int) ([]*MealFood, error) {
	// First, get all the food IDs for the given meal.
	var foodIDs []int
	query := `SELECT food_id FROM meal_foods WHERE meal_id = $1`
	err := tx.Select(&foodIDs, query, mealID)
	if err != nil {
		return nil, err
	}

	// Now, for each food ID, get the full food details and preferences.
	var mealFoods []*MealFood
	for _, foodID := range foodIDs {
		mf, err := getMealFoodWithPref(tx, foodID, int64(mealID))
		if err != nil {
			return nil, err
		}
		mealFoods = append(mealFoods, mf)
	}

	return mealFoods, nil
}

// getMealFoodWithPref retrieves one of the foods for a given meal,
// along its preferences.
// Nutrients are for portion size (100 serving unit)
func getMealFoodWithPref(tx *sqlx.Tx, foodID int, mealID int64) (*MealFood, error) {
	mf := MealFood{}

	// Get the food details
	err := tx.Get(&mf.Food, "SELECT * FROM foods WHERE food_id = ?", foodID)
	if err != nil {
		log.Println("Failed to get food.")
		return nil, err
	}

	// Get the serving size and number of servings, preferring meal_food_prefs and then food_prefs and then default
	query := `
        SELECT
            COALESCE(mfp.serving_size, fp.serving_size, f.serving_size) AS serving_size,
            COALESCE(mfp.number_of_servings, fp.number_of_servings, 1) AS number_of_servings,
						CASE WHEN mfp.serving_size IS NOT NULL OR fp.serving_size IS NOT NULL THEN TRUE ELSE FALSE END as has_preference
        FROM foods f
        LEFT JOIN meal_food_prefs mfp ON mfp.food_id = f.food_id AND mfp.meal_id = $1
        LEFT JOIN food_prefs fp ON fp.food_id = f.food_id
        WHERE f.food_id = $2
        LIMIT 1
    `

	// Execute the SQL query and assign the result to the MealFood struct
	err = tx.Get(&mf, query, mealID, foodID)
	if err != nil {
		log.Println("Failed to select serving size and number of servings.")
		return nil, err
	}

	// Execute the SQL query and assign the result to the calories field
	// in the MealFood struct
	err = tx.Get(&mf.Food.PortionCals, "SELECT amount FROM food_nutrients WHERE food_id = ? AND nutrient_id IN (SELECT nutrient_id FROM nutrients WHERE nutrient_name = 'Energy' AND unit_name = 'KCAL' LIMIT 1)", foodID)
	if err != nil {
		log.Println("Failed to select portion calories.")
		return nil, err
	}

	// Get the macros for the food
	mf.Food.FoodMacros, err = getFoodMacros(tx, foodID)
	if err != nil {
		log.Println("Failed to get food macros.")
		return nil, err
	}

	// If a preference was found (either in meal_food_prefs or food_prefs),
	// adjust nutrient values based on the serving size and number of servings.
	if mf.HasPreference {
		// 100 is for porition size which nutrient amounts represent.
		ratio := mf.ServingSize / 100
		mf.Food.PortionCals *= ratio * mf.NumberOfServings
		mf.Food.FoodMacros.Protein *= ratio * mf.NumberOfServings
		mf.Food.FoodMacros.Fat *= ratio * mf.NumberOfServings
		mf.Food.FoodMacros.Carbs *= ratio * mf.NumberOfServings
	}

	return &mf, nil
}

// printMealDetails prints the foods that make up the meal and their preferences.
func printMealDetails(mealFoods []*MealFood) {
	for i, mf := range mealFoods {
		fmt.Printf("[%d] ", i+1)
		printMealFood(mf)
	}
}

// printMealFood prints details of a given MealFood object.
func printMealFood(mealFood *MealFood) {
	fmt.Printf("%s\n", mealFood.Food.Name)
	//871682fmt.Printf("\t%.2f %s\n", math.Round(100*mealFood.NumberOfServings*mealFood.ServingSize)/100, mealFood.Food.ServingUnit)
	fmt.Printf("Serving Size: %.2f\n", mealFood.ServingSize)
	fmt.Printf("Number of Servings: %.2f\n", mealFood.NumberOfServings)
	fmt.Printf("Calories: %.2f\n", mealFood.Food.PortionCals)

	fmt.Println("Macros:")
	fmt.Printf("  - Protein: %.2f\n", mealFood.Food.FoodMacros.Protein)
	fmt.Printf("  - Fat: %.2f\n", mealFood.Food.FoodMacros.Fat)
	fmt.Printf("  - Carbs: %.2f\n", mealFood.Food.FoodMacros.Carbs)
}

// promptUserEditDecision prompts the user to select one of foods that
// make up a meal to edit or <enter> to use existing values.
func promptUserEditDecision() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter index of food to edit or press <enter> for existing values: ")
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("promptUserEditDecision: %v\n", err)
	}
	// Remove the newline character at the end of the string
	response = strings.TrimSpace(response)
	return response
}

// addMealEntry inserts a meal entry into the database.
func addMealEntry(tx *sqlx.Tx, meal Meal, date time.Time) error {
	const query = `
    INSERT INTO daily_meals (meal_id, date, time)
    VALUES ($1, $2, $3)
    `

	_, err := tx.Exec(query, meal.ID, date.Format(dateFormat), date.Format(dateFormatTime))
	if err != nil {
		return err
	}

	// If there was an error executing the query, return the error
	if err != nil {
		return fmt.Errorf("addMealEntry: %w", err)
	}

	return nil
}

// addMealFoodEntries bulk inserts foods that make up the meal into the database.
func addMealFoodEntries(tx *sqlx.Tx, mealID int, mealFoods []*MealFood, date time.Time) error {
	// Prepare a statement for bulk insert
	stmt, err := tx.Preparex("INSERT INTO daily_foods (food_id, meal_id, date, time, serving_size, number_of_servings) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Iterate over each food and insert into the database
	for _, mf := range mealFoods {
		_, err = stmt.Exec(mf.Food.ID, mealID, date.Format(dateFormat), date.Format(dateFormatTime), mf.ServingSize, mf.NumberOfServings)
		if err != nil {
			return err
		}
	}

	return nil
}

// FoodLogSummary fetches and prints a food log summary.
func FoodLogSummary(db *sqlx.DB) error {
	// Start a new transaction
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	// Get total amonut of foods logged in the database.
	total, err := getTotalFoodsLogged(tx)
	fmt.Printf("\nTotal Foods Logged: %d\n", total)

	// Get most frequently consumed foods.
	foods, err := getFrequentFoods(tx, 10)
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

// getTotalFoodsLogged fetches and returns the total amount of foods
// entries logged in the database.
func getTotalFoodsLogged(tx *sqlx.Tx) (int, error) {
	var totalFoodsLogged int
	query := `
      SELECT COUNT(*)
      FROM daily_foods
    `
	if err := tx.Get(&totalFoodsLogged, query); err != nil {
		return 0, err
	}

	return totalFoodsLogged, nil
}

// getFrequentFoods retrieves most recently logged food entries.
func getFrequentFoods(tx *sqlx.Tx, limit int) ([]DailyFoodCount, error) {
	// Since DailyFood struct does not currently support time field, the
	// query excludes the time field from the selected records.
	const query = `
        SELECT df.food_id, df.date, df.serving_size, df.number_of_servings, f.food_name, f.serving_unit, COUNT(*) as count
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

// checkInput checks if the user input is positive
func checkInput(n float64) error {
	if n < 0 {
		return errors.New("invalid number")
	}
	return nil
}

// promptCals prompts the user to enter caloric intake for the previous
// day.
func promptCals() (calories float64, err error) {
	fmt.Print("Enter caloric intake for the day: ")
	fmt.Scanln(&calories)

	return calories, checkInput(calories)
}

/*
// Log appends a new entry to the csv file passed in as an agurment.
func Log(u *UserInfo, s string) error {
	var date time.Time
	var err error

	// Get user weight.
	u.Weight, err = getWeight(u.System)
	if err != nil {
		return err
	}

	// Get user calories for the day.
	cals, err := promptCals()
	if err != nil {
		return err
	}

	for {
		// Prompt user entry date.
		r := promptDate("Enter entry date (YYYY-MM-DD) [Press Enter for today's date]: ")

		// If user entered default date,
		if r == "" {
			// set date to today's date.
			date = time.Now()
			break
		}

		// Check if date is a valid date.
		date, err = validateDateStr(r)
		if err != nil {
			fmt.Printf("%v. Please try again.\n", err)
			continue
		}

		// Validate the date is not in the past.
		if date.Before(time.Now()) {
			fmt.Println("The entered date is in the past. Please enter today's date or a future date.")
			continue
		}

		break
	}

	// Save updated user info.
	err = saveUserInfo(tx, u)
	if err != nil {
		return err
	}

	// Open file for append.
	f, err := os.OpenFile(s, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	if err != nil {
		log.Println(err)
		return err
	}

	// Append user calorie input to csv file.
	line := fmt.Sprintf("%.2f,%.2f,%s\n", u.Weight, cals, date.Format(dateFormat))
	_, err = f.WriteString(line)
	if err != nil {
		log.Println(err)
		return err
	}

	fmt.Println("Added entry.")

	return nil
}
*/

/*

// Subset creates and returns and array containing the
// valid log entries.
//
// Assumptions:
// * Diet phase activity has been checked. That is, this function should
// not be called for a diet phase that is not currently active.

// subset returns a subset of the dataframe containing the entries that
// were logged during an active diet phase.
func Subset(logs *dataframe.DataFrame, indices []int) *dataframe.DataFrame {
	s1 := dataframe.NewSeriesString("weight", nil)
	s2 := dataframe.NewSeriesString("calories", nil)
	s3 := dataframe.NewSeriesString("date", nil)
	s := dataframe.NewDataFrame(s1, s2, s3)

	for _, idx := range indices {
		row := logs.Row(idx, false, dataframe.SeriesIdx|dataframe.SeriesName)
		s.Append(nil, row)
	}

	return s
}
*/

// getValidLog creates and returns array containing the
// valid log entries.
//
// Assumptions:
// * Diet phase activity has been checked. That is, this function should
// not be called for a diet phase that is not currently active.
func GetValidLog(u *UserInfo, entries *[]Entry) *[]Entry {
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

/*
// getValidLogIndices creates and returns and int array containing the
// indices of the the valid log entries.
//
// Assumptions:
// * Diet phase activity has been checked. That is, this function should
// not be called for a diet phase that is not currently active.
func GetValidLogIndices(u *UserInfo, logs *dataframe.DataFrame) []int {
	today := time.Now()

	var validIndices []int
	for i := 0; i < logs.NRows(); i++ {
		date, err := time.Parse(dateFormat, logs.Series[dateCol].Value(i).(string))
		if err != nil {
			log.Println("ERROR: Couldn't parse date:", err)
			return nil
		}

		// Only consider dates that fall somewhere inbetween the diet
		// start date and the current date.
		if (date.After(u.Phase.StartDate) || isSameDay(date, u.Phase.StartDate)) && (date.Before(today) || isSameDay(date, today)) {
			validIndices = append(validIndices, i)
		}
	}

	return validIndices
}
*/
