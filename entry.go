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
	fullBlock         = "\u2588"
	lightBlock        = "\u2592"
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

// GetAllEntries returns all the user's entries from the database.
func GetAllEntries(db *sqlx.DB) (*[]Entry, error) {
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
		log.Fatalf("GetAllEntries: %v\n", err)
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
func LogWeight(u *UserInfo, db *sqlx.DB) {
	for {
		// Get weight from user
		weight, err := getWeight(u.System)
		if err != nil {
			fmt.Printf("%v. Please try again.\n", err)
			continue
		}

		// Get weight entry date from user
		date := getDateNotPast("Enter weight entry date")

		if err = addWeightEntry(db, date, weight); err != nil {
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

	fmt.Println("Successfully added weight entry.")
	return nil
}

// getDateNotPast prompts user for date that it not in the past, validates user
// response until user enters a valid date, and return the valid date.
func getDateNotPast(s string) (date time.Time) {
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
	response := promptSelectEntry("Enter entry index to select or date to search (YYYY-MM-DD): ")
	idx, err := strconv.Atoi(response)

	// While response is an integer
	for err == nil {
		// If integer is invalid,
		if 1 > idx || idx > len(log) {
			fmt.Println("Number must be between 0 and number of entries. Please try again.")
			response = promptSelectEntry("Enter entry index to select or date to search (YYYY-MM-DD): ")
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
func promptSelectEntry(s string) string {
	reader := bufio.NewReader(os.Stdin)
	//fmt.Printf("Enter entry index to select or date to search (YYYY-MM-DD): ")
	fmt.Printf("%s: ", s)
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

	// Get selected food
	food, err := selectFood(tx)
	if err != nil {
		if errors.Is(err, ErrDone) {
			fmt.Println("No food selected.")
			return nil // Not really an "error" situation
		}
		log.Println(err)
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

	// Get food with up to date food preferences.
	foodWithPref, err := getFoodWithPref(tx, food.ID)
	if err != nil {
		return err
	}

	// Get date of food entry.
	date := getDateNotPast("Enter food entry date")

	// Log selected food to the food log database table. Taking into
	// account food preferences.
	err = addFoodEntry(tx, foodWithPref, date)
	if err != nil {
		log.Println(err)
		return err
	}
	fmt.Println("Successfully added food entry.")

	return tx.Commit()
}

// selectFood prompts the user to enter a search term, prints the matched
// foods, prompts user to enter an index to select a food or another
// serach term for a different food. This repeats until user enters a
// valid index.
func selectFood(tx *sqlx.Tx) (Food, error) {
	fmt.Println("Recently logged foods:")

	// Get most recently logged foods.
	recentFoods, err := getRecentlyLoggedFoods(tx, searchLimit)
	if err != nil {
		log.Println(err)
		return Food{}, err
	}

	for i, food := range recentFoods {
		fmt.Printf("[%d] %s\n", i+1, food.Name)
	}

	// Get response.
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
			brandDetail := ""
			if food.BrandName != "" {
				brandDetail = " (Brand: " + food.BrandName + ")"
			}
			fmt.Printf("[%d] %s%s\n", i+1, food.Name, brandDetail)
		}

		/*
			// Print foods.
			for i, food := range *filteredFoods {
				fmt.Printf("[%d] %s\n", i+1, food.Name)
			}
		*/

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

// getRecentlyLoggedFoods retrieves most recently logged foods.
func getRecentlyLoggedFoods(tx *sqlx.Tx, limit int) ([]Food, error) {
	const query = `
  SELECT f.*
  FROM (
    SELECT *, ROW_NUMBER() OVER (PARTITION BY food_id ORDER BY date DESC) AS rn
    FROM daily_foods
  ) AS df
  INNER JOIN foods f ON df.food_id = f.food_id
  WHERE df.rn = 1
  ORDER BY df.date DESC
  LIMIT $1;
`

	var foods []Food
	if err := tx.Select(&foods, query, limit); err != nil {
		return nil, err
	}

	return foods, nil
}

// searchFoods searchs through all foods and returns food that contain
// the search term.
func searchFoods(tx *sqlx.Tx, response string) (*[]Food, error) {
	var foods []Food

	// Prioritize exact match, then match foods where `food_name` starts
	// with the search term, and finally any foods where the `food_name`
	// contains the search term.
	/*
		SELECT f.* FROM foods f INNER JOIN foods_fts ff ON ff.food_id = f.food_id WHERE foods_fts MATCH 'ribeye steak' ORDER BY bm25(foods_fts) LIMIT 20;
	*/
	query := `
				SELECT f.*
				FROM
				foods f
				INNER JOIN foods_fts s ON s.food_id = f.food_id
				WHERE foods_fts MATCH $1
				ORDER BY bm25(foods_fts)
        LIMIT $2`

	/*
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
	*/
	if err := tx.Select(&foods, query, response, searchLimit); err != nil {
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
	const query = `
    SELECT
      f.food_id,
			f.serving_size AS default_serving_size,
      COALESCE(fp.serving_size, 100) AS serving_size,
			f.household_serving,
			COALESCE(fp.number_of_servings, 1) AS number_of_servings,
			f.serving_unit
    FROM foods f
    LEFT JOIN food_prefs fp ON f.food_id = fp.food_id
    WHERE f.food_id = $1
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
	householdStr := ""
	if pref.HouseholdServing != "" {
		householdStr = fmt.Sprintf("(%s)", pref.HouseholdServing)
	}
	fmt.Printf("Suggested Serving Size: %.2f %s %s\n", pref.DefaultServingSize, pref.ServingUnit, householdStr)
	fmt.Printf("Current Serving Size: %.2f %s\n", math.Round(100*pref.ServingSize)/100, pref.ServingUnit)
	fmt.Printf("Number of Servings: %.1f\n", math.Round(10*pref.NumberOfServings)/10)
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
func addFoodEntry(tx *sqlx.Tx, f *Food, date time.Time) error {
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

	// Make database update for food preferences.
	if err := updateFoodPrefs(tx, pref); err != nil {
		return err
	}

	// Get food with up to date food preferences.
	foodWithPref, err := getFoodWithPref(tx, entry.FoodID)
	if err != nil {
		return err
	}

	// Update food entry.
	err = updateFoodEntry(tx, entry.ID, *foodWithPref)
	if err != nil {
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
	recentFoods, err := getRecentFoodEntries(tx, searchLimit)
	if err != nil {
		log.Println(err)
		return DailyFood{}, err
	}

	// Print recent food entries.
	printFoodEntries(recentFoods)

	// Get response.
	response := promptSelectEntry("Enter entry index to select or date to search (YYYY-MM-DD): ")
	idx, err := strconv.Atoi(response)

	// While response is an integer
	for err == nil {
		// If integer is invalid,
		if 1 > idx || idx > len(recentFoods) {
			fmt.Println("Number must be between 0 and number of entries. Please try again.")
			response = promptSelectEntry("Enter entry index to select or date to search (YYYY-MM-DD): ")
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
			response = promptSelectEntry("Enter entry index to select or date to search (YYYY-MM-DD): ")
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
			response = promptSelectEntry("Enter entry index to select or date to search (YYYY-MM-DD): ")
			continue
		}

		// Print the foods entries for given date.
		printFoodEntries(filteredEntries)

		response = promptSelectEntry("Enter entry index to select or date to search (YYYY-MM-DD): ")
		idx, err := strconv.Atoi(response)

		// While response is an integer
		for err == nil {
			// If integer is invalid,
			if idx != 1 {
				fmt.Println("Number must be between 0 and number of entries. Please try again.")
				response = promptSelectEntry("Enter entry index to select or date to search (YYYY-MM-DD): ")
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
  SELECT df.id, df.food_id, df.meal_id, df.date, df.serving_size,
	df.number_of_servings, f.food_name, f.serving_unit
  FROM (
    SELECT *, ROW_NUMBER() OVER (PARTITION BY food_id ORDER BY date DESC) AS rn
    FROM daily_foods
  ) AS df
  INNER JOIN foods f ON df.food_id = f.food_id
  WHERE df.rn = 1
  ORDER BY df.date DESC
  LIMIT $1;
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
        SELECT df.id, df.food_id, df.meal_id, df.date, df.serving_size,
				df.number_of_servings, f.food_name, f.serving_unit
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
func updateFoodEntry(tx *sqlx.Tx, entryID int, f Food) error {
	const query = `
        UPDATE daily_foods
        SET serving_size = $1, number_of_servings = $2, calories = $3,
				protein = $4, fat = $5, carbs = $6, price = $7
        WHERE id = $8
    `

	// Execute the update statement
	_, err := tx.Exec(query, f.ServingSize, f.NumberOfServings, f.Calories,
		f.FoodMacros.Protein, f.FoodMacros.Fat, f.FoodMacros.Carbs, f.Price, entryID)

	// If there was an error executing the query, return the error
	if err != nil {
		log.Println("Failed to update entry in daily_foods.")
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
		fmt.Printf("- %s: %.1f %s x %.1f serving | %.0f cals |\n",
			entry.FoodName, entry.ServingSize, entry.ServingUnit,
			entry.NumberOfServings, entry.Calories)
	}

	return tx.Commit()
}

// getAllFoodEntries retrieves all logged food entries. Ordered by most
// most recent date.
func getAllFoodEntries(tx *sqlx.Tx) ([]DailyFood, error) {
	// Since DailyFood struct does not currently support time field, the
	// queury excludes the time field from the selected records.
	const query = `
        SELECT 
				df.id, df.food_id, df.meal_id, df.date, df.serving_size,
				df.number_of_servings, df.calories, df.price, f.food_name,
				f.serving_unit
        FROM daily_foods df
        INNER JOIN foods f ON df.food_id = f.food_id
        ORDER BY df.date DESC
    `
	var entries []DailyFood
	if err := tx.Select(&entries, query); err != nil {
		log.Println("Failed to get main details from daily food entries.")
		return nil, err
	}

	const macrosQuery = `
    SELECT protein, fat, carbs
    FROM daily_foods
    WHERE id = $1
`

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

		// If user pressed <Enter>, break the loop.
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

	fmt.Println("Successfully added meal entry.")

	return tx.Commit()
}

// selectMeal prints the user's meals, prompts them to select a meal,
// and returns the selected meal.
func selectMeal(tx *sqlx.Tx) (Meal, error) {
	// Get recently logged meals
	meals, err := getMealsWithRecentFirst(tx)
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

// getMealsWithRecentFirst retrieves the meals that have been logged
// recently first and then retrieves the remaining meals.
func getMealsWithRecentFirst(tx *sqlx.Tx) ([]Meal, error) {
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
	if err := tx.Select(&meals, query); err != nil {
		return nil, err
	}

	return meals, nil
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
func getMealFoodWithPref(tx *sqlx.Tx, foodID int, mealID int64) (*MealFood, error) {
	mf := MealFood{}

	// Get the food details
	err := tx.Get(&mf.Food, "SELECT * FROM foods WHERE food_id = $1", foodID)
	if err != nil {
		log.Println("Failed to get food.")
		return nil, err
	}

	// Get the serving size and number of servings, preferring meal_food_prefs and then food_prefs and then default
	query := `
        SELECT
            COALESCE(mfp.serving_size, fp.serving_size, 100) AS serving_size,
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
	err = tx.Get(&mf.Food.Calories, "SELECT amount FROM food_nutrients WHERE food_id = ? AND nutrient_id IN (SELECT nutrient_id FROM nutrients WHERE nutrient_name = 'Energy' AND unit_name = 'KCAL' LIMIT 1)", foodID)
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
		mf.Food.Calories *= ratio * mf.NumberOfServings
		mf.Food.FoodMacros.Protein *= ratio * mf.NumberOfServings
		mf.Food.FoodMacros.Fat *= ratio * mf.NumberOfServings
		mf.Food.FoodMacros.Carbs *= ratio * mf.NumberOfServings
		mf.Food.Price *= ratio * mf.NumberOfServings
	}

	return &mf, nil
}

// getFoodWithPref retrieves one food, along its preferences.
func getFoodWithPref(tx *sqlx.Tx, foodID int) (*Food, error) {
	f := Food{}

	// Get the food details.
	err := tx.Get(&f, "SELECT * FROM foods WHERE food_id = $1", foodID)
	if err != nil {
		log.Println("Failed to get food.")
		return nil, err
	}

	// Override existing serving size and number of servings if there
	// exists a matching entry in the food_prefs table for the food id.
	query := `
        SELECT
            COALESCE(fp.serving_size, 100) AS serving_size,
            COALESCE(fp.number_of_servings, 1) AS number_of_servings,
            CASE WHEN fp.serving_size IS NOT NULL THEN TRUE ELSE FALSE END as has_preference
        FROM foods f
        LEFT JOIN food_prefs fp ON fp.food_id = f.food_id
        WHERE f.food_id = $1
        LIMIT 1
    `

	// Execute the SQL query and assign the result to the Food struct
	err = tx.Get(&f, query, foodID)
	if err != nil {
		log.Println("Failed to select serving size and number of servings.")
		return nil, err
	}

	// Execute the SQL query and assign the result to the calories field
	// in the Food struct
	err = tx.Get(&f.Calories, "SELECT amount FROM food_nutrients WHERE food_id = ? AND nutrient_id IN (SELECT nutrient_id FROM  nutrients WHERE nutrient_name = 'Energy' AND unit_name = 'KCAL' LIMIT 1)", foodID)
	if err != nil {
		log.Println("Failed to select portion calories.")
		return nil, err
	}

	// Get the macros for the food.
	f.FoodMacros, err = getFoodMacros(tx, foodID)
	if err != nil {
		log.Println("Failed to get food macros.")
		return nil, err
	}

	// If a preference was found (either in food_prefs),
	// adjust nutrient values based on the serving size and number of servings.
	if f.HasPreference {
		// 100 is for porition size which nutrient amounts represent.
		ratio := f.ServingSize / 100
		f.Calories *= ratio * f.NumberOfServings
		f.FoodMacros.Protein *= ratio * f.NumberOfServings
		f.FoodMacros.Fat *= ratio * f.NumberOfServings
		f.FoodMacros.Carbs *= ratio * f.NumberOfServings
		f.Price *= ratio * f.NumberOfServings
	}

	return &f, nil
}

// printMealDetails prints the foods that make up the meal and their preferences.
func printMealDetails(mealFoods []*MealFood) {
	var priceTotal float64
	for i, mf := range mealFoods {
		fmt.Printf("[%d] ", i+1)
		printMealFood(mf)
		priceTotal += mf.Food.Price
	}
	fmt.Printf("Total estimated cost of meal: $%.2f\n", priceTotal)
}

// printMealFood prints details of a given MealFood object.
func printMealFood(mealFood *MealFood) {
	fmt.Printf("%s: %.2f %s x %.2f serving, %.2f cals ($%.2f)\n",
		mealFood.Food.Name, mealFood.ServingSize, mealFood.Food.ServingUnit,
		mealFood.NumberOfServings, mealFood.Food.Calories, mealFood.Food.Price)

	fmt.Printf("\tMacros: | Protein: %-3.2fg | Carbs: %-3.2fg | Fat: %-3.2fg |\n", mealFood.Food.FoodMacros.Protein, mealFood.Food.FoodMacros.Carbs, mealFood.Food.FoodMacros.Fat)
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

// FoodLogSummaryDay prints the current nutritional totals for a given
// day and provides insight on progress towards nutritional goals.
func FoodLogSummaryDay(db *sqlx.DB, u *UserInfo) error {
	// Start a new transaction
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	// Get the food entries for the present day.
	entries, err := getFoodEntriesForDate(tx, time.Now())
	if err != nil {
		return err
	}

	// TODO: if there are zero entries for today, then return early.

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

// getFoodEntriesForDate retrieves the food entries for a given date.
func getFoodEntriesForDate(tx *sqlx.Tx, date time.Time) ([]DailyFood, error) {
	// Since DailyFood struct does not currently support time field, the
	// queury excludes the time field from the selected records.
	const query = `
        SELECT df.id, df.food_id, df.meal_id, df.date, df.serving_size,
				df.number_of_servings, df.calories, df.price, f.food_name, f.serving_unit
        FROM daily_foods df
        INNER JOIN foods f ON df.food_id = f.food_id
				WHERE date = $1
        ORDER BY df.date DESC
    `

	var entries []DailyFood
	if err := tx.Select(&entries, query, date.Format(dateFormat)); err != nil {
		log.Printf("Failed to get main details from daily food entries: %v\n", err)
		return nil, err
	}

	const macrosQuery = `
    SELECT protein, fat, carbs
    FROM daily_foods
    WHERE id = $1
	`

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

// printNutrientProgress prints the nutrient progress
func printNutrientProgress(current, goal float64, name string) {
	progressBar := renderProgressBar(current, goal)
	fmt.Printf("%-9s %s %3.0f%% (%.0fg / %.0fg)\n", name+":", progressBar, current*100/goal, current, goal)
}

// printCalorieProgress prints the calories progress
func printCalorieProgress(current, goal float64, name string) {
	progressBar := renderProgressBar(current, goal)
	fmt.Printf("%-9s %s %3.0f%% (%.0f / %.0f)\n", name+":", progressBar, current*100/goal, current, goal)
}

// renderProgressBar renders an ASCII progress bar
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

// GetValidLog creates and returns array containing the
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
