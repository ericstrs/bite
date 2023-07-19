package calories

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

const (
	selectNutrientAmountSQL = "SELECT amount FROM food_nutrients WHERE food_id = ? AND nutrient_id = ?"
	selectNutrientIdSQL     = "SELECT nutrient_id FROM nutrients WHERE nutrient_name = ? LIMIT 1"
	searchLimit             = 25
)

type Meal struct {
	ID        int    `db:"meal_id"`
	Name      string `db:"meal_name"`
	Frequency int
	// TODO: adds fields or new meal_food struct?
}

type Food struct {
	ID               int     `db:"food_id"`
	Name             string  `db:"food_name"`
	ServingUnit      string  `db:"serving_unit"`
	ServingSize      float64 `db:"serving_size"`
	HouseholdServing string  `db:"household_serving"`
	PortionCals      float64
	Macros           *Macros
}

type FoodPref struct {
	FoodId           int     `db:"food_id"`
	NumberOfServings float64 `db:"number_of_servings"`
	ServingSize      float64 `db:"serving_size"`
}

type MealFoodPref struct {
	FoodId           int     `db:"food_id"`
	MealId           int     `db:"meal_id"`
	NumberOfServings float64 `db:"number_of_servings"`
	ServingSize      float64 `db:"serving_size"`
}

type Macros struct {
	Protein float64
	Fat     float64
	Carbs   float64
}

const usage = "Usage: ./m [add|display]"

func promptDeleteMeal(db *sqlx.DB) {
	// Get all meals
	meals, err := getAllMeals(db)
	if err != nil {
		log.Fatal(err)
	}
	m := selectMeal(db, meals)
	deleteMeal(db, m.ID)
}

// selectMeal prints the user's meals, prompts them to select a meal,
// and returns the selected meal.
func selectMeal(db *sqlx.DB, meals []Meal) Meal {
	for {
		// Print meals.
		for i, meal := range meals {
			fmt.Printf("[%d] %s\n", i+1, meal.Name)
		}

		response := getSelectResponse(len(meals), "meal")

		idx, err := strconv.Atoi(response)
		// If the user's input was a number,
		if err == nil {
			return meals[idx-1]
		}
		// Otherwise, the user's input was not a number

		// Get the filtered meals
		filteredMeals := searchMeals(db, meals, response)
		// If there were no matchs found,
		if len(filteredMeals) == 0 {
			fmt.Println("No matches found. Please try again.")
			continue
		}
		// Otherwise, update meals to the filtered meals
		meals = filteredMeals
		// TODO: This is not ideal. Updating to filtered meals will mean
		// that the printed options will get smaller and smaller.
		// Solution: search terms print idxs starting from where all meal
		// print left off.
		// Pro: Lets the user not have to restart the process over again
		// when they searched so much that there are very little meals to
		// print.
		// Con: User may have to type long numbers.
	}
}

// selectFood prints the foods, prompts user to select a food, and
// returns the selected food.
//
// TODO
// * Simplify code.
func selectFood(db *sqlx.DB) Food {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter food name: ")
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	// Remove the newline character at the end of the string
	response = strings.TrimSpace(response)

	for {
		// Get the filtered foods
		filteredFoods, err := searchFoods(db, response)
		if err != nil {
			log.Fatal(err)
		}

		// If no matches found,
		if len(*filteredFoods) == 0 {
			fmt.Println("No matches found. Please try again.")
			reader := bufio.NewReader(os.Stdin)
			fmt.Printf("Enter food name: ")
			response, err = reader.ReadString('\n')
			if err != nil {
				log.Fatal(err)
			}
			// Remove the newline character at the end of the string
			response = strings.TrimSpace(response)
			continue
		}

		// Print foods.
		for i, food := range *filteredFoods {
			fmt.Printf("[%d] %s\n", i+1, food.Name)
		}

		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Enter either of the food or a search term: ")
		response, err = reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		// Remove the newline character at the end of the string
		response = strings.TrimSpace(response)

		idx, err := strconv.Atoi(response)
		// While response is a number
		for err == nil {
			// If number is invalid,
			if 1 > idx || idx > len(*filteredFoods) {
				fmt.Println("Number must be between 0 and number of foods. Please try again.")
				// prompt for response.
				reader := bufio.NewReader(os.Stdin)
				fmt.Printf("Enter either of the food or a search term: ")
				response, err = reader.ReadString('\n')
				if err != nil {
					log.Fatal(err)
				}
				// Remove the newline character at the end of the string
				response = strings.TrimSpace(response)

				idx, err = strconv.Atoi(response)
				continue
			}
			// Otherwise, return food at valid index.
			return (*filteredFoods)[idx-1]
		}
	}
	// Otherwise, the user's response was a search term
}

// getSelectResponse prompts user to select an item (meal or food),
// validates their response until they've entered a valid response,
// and returns the valid response.
func getSelectResponse(itemsLen int, itemName string) (r string) {
	for {
		r := promptSelectResponse(itemName)
		err := validateSelectResponse(r, itemName, itemsLen)
		if err != nil {
			fmt.Printf("Invalid input: %v. Please try again.\n", err)
			continue
		}
		break
	}
	return r
}

// promptSelectResponse returns meal to select or a search term.
func promptSelectResponse(item string) (s string) {
	fmt.Printf("Enter either the index of the %s to select or a search term: ", item)
	_, err := fmt.Scanln(&s)
	if err != nil {
		log.Fatal(err)
	}
	return s
}

// validateSelectResponse validates meal to select.
func validateSelectResponse(s, item string, itemLen int) error {
	idx, err := strconv.Atoi(s)
	// If the user's input was a number,
	if err == nil {
		if 1 > idx || idx > itemLen {
			return fmt.Errorf("Number must be between 0 and number of %ss.", item)
		}
	}

	return nil
}

func promptDeleteResponse() (r string) {
	// Prompt for user input
	fmt.Printf("Enter the index of the meal to delete or a search term: ")
	_, err := fmt.Scanln(&r)
	if err != nil {
		log.Fatal(err)
	}
	return r
}

// searchMeals searches through meals slice and returns meals that
// contain the search term.
func searchMeals(db *sqlx.DB, meals []Meal, term string) []Meal {
	filteredMeals := []Meal{}
	for _, meal := range meals {
		if strings.Contains(strings.ToLower(meal.Name), strings.ToLower(term)) {
			filteredMeals = append(filteredMeals, meal)
		}
	}
	return filteredMeals
}

// searchFoods searchs through all foods and returns food that contain
// the search term.
func searchFoods(db *sqlx.DB, response string) (*[]Food, error) {
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
	err := db.Select(&foods, query, "%"+response+"%", response, response+"%", searchLimit)
	if err != nil {
		log.Printf("Search for foods failed: %v\n", err)
		return nil, err
	}

	return &foods, nil
}

func promptMeal() (m string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Enter the name of your new meal: ")
	m, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	return strings.TrimSpace(m)
}

func addMeal(db *sqlx.DB, meal Meal) {
	// Insert the new meal into the database
	_, err := db.Exec(`INSERT INTO meals (name, frequency) VALUES (?, ?)`, meal.Name, meal.Frequency)
	if err != nil {
		log.Fatal("ERROR: ", err)
	}
}

func getAllMeals(db *sqlx.DB) ([]Meal, error) {
	m := []Meal{}
	err := db.Select(&m, "SELECT meal_id, meal_name FROM meals")
	if err != nil {
		return nil, err
	}

	return m, nil
}

/*
func getAllMealsSortedByFreq(db *sqlx.DB) ([]Meal, error) {
	m := []Meal{}
	err := db.Select(&m, "SELECT id, name, frequency FROM meals ORDER BY frequency DESC")
	if err != nil {
		return nil, err
	}

	return m, nil
}
*/

// TODO:
// * Grab foods that make up the meal.
// * For serving size and number of serving, first check `food_prefs`
// table for entries.
// getOneMeal retrieves the details for a given meal.
func getOneMeal(db *sqlx.DB, mealID int) (*Meal, error) {
	m := Meal{}
	err := db.Get(&m, "SELECT id, name, frequency FROM meals WHERE id=?", mealID)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

func printMeals(m []Meal) {
	// Print the data
	for _, meal := range m {
		fmt.Printf("ID: %d, Name: \"%s\", Frequency: %d\n", meal.ID, meal.Name, meal.Frequency)
	}
}

func displayAllMeals(db *sqlx.DB) {
	m, err := getAllMeals(db)
	if err != nil {
		log.Fatal(err)
		return
	}

	printMeals(m)
}

func displayOneMeal(db *sqlx.DB, mealID int) {
	m, err := getOneMeal(db, mealID)
	if err != nil {
		log.Fatal(err)
		return
	}

	// TODO: update print to include rest of the fields.
	fmt.Printf("ID: %d, Name: %s\n", m.ID, m.Name)
}

func deleteMeal(db *sqlx.DB, mealID int) error {
	_, err := db.Exec("DELETE FROM meals WHERE id=?", mealID)
	return err
}

func incrementMealFrequency(db *sqlx.DB, mealID int) error {
	_, err := db.Exec(`UPDATE meals SET frequency = frequency + 1 WHERE id=?`, mealID)
	return err
}

// TODO: function to get the slice `food_id`'s specifed by search term.
// Then iterate over slice calling to getOneFood for each iteration.

// TODO: displayFood takes in a slice of meal id's to prints its contents.

// getFoodPref prompts user for food perferences, validates their
// response until they've entered a valid response, and returns the
// valid response.
//
// TODO:
// * Let user press <enter> to keep old values.
// * Would be nice to see manufacturer's values.
func getFoodPref(foodID int) *FoodPref {
	pref := &FoodPref{}

	pref.FoodId = foodID

	pref.ServingSize = getServingSize()

	pref.NumberOfServings = getNumServings()

	return pref
}

// getServingSize prompts user for serving size, validates their
// response until they've entered a valid response, and returns the
// valid response.
func getServingSize() float64 {
	fmt.Print("Enter new serving size: ")

	var servingSize float64
	for {
		// Prompt for serving size.
		servingSize, err := promptServingSize()
		if err != nil {
			fmt.Println("Invalid serving size: Serving size must be a number. Please try again.")
			continue
		}

		// Validate user response.
		err = validateServingSize(servingSize)
		if err != nil {
			fmt.Printf("Invalid serving size: %v. Please try again.", err)
			continue
		}

		break
	}
	return servingSize
}

// promptServingSize prompts and returns serving size.
func promptServingSize() (s float64, err error) {
	_, err = fmt.Scan(&s)
	if err != nil {
		return 0, err
	}
	return s, nil
}

// validateServingSize validates serving size.
func validateServingSize(servingSize float64) error {
	if 0 <= servingSize || servingSize > 1000 {
		return fmt.Errorf("Serving size must be between 0 and 1000.")
	}
	return nil
}

// getNumServings prompts user for number of servings, validates their
// response until they've entered a valid response, and returns the
// valid response.
func getNumServings() float64 {
	fmt.Print("Enter new number of servings: ")
	var numOfServings float64
	for {
		// Prompt user for number of servings.
		s, err := promptNumServings()
		if err != nil {
			fmt.Println("Invalid number of servings: Number of servings must be a number. Please try again.")
			continue
		}

		// Validate user response.
		err = validateNumServings(s)
		if err != nil {
			fmt.Printf("Invalid number of servings: %v. Please try again.", err)
			continue
		}
		break
	}
	return numOfServings
}

// promptNumServings prompts and returns number of servings.
func promptNumServings() (s float64, err error) {
	_, err = fmt.Scan(&s)
	if err != nil {
		return 0, err
	}
	return s, nil
}

// validateNumServings validates the number of servings.
func validateNumServings(numServings float64) error {
	if 0 <= numServings || numServings > 100 {
		return fmt.Errorf("Number of servings must be between 0 and 100.")
	}
	return nil
}

// updateFoodPrefs updates the user's preferences for a given
// food.
func updateFoodPrefs(db *sqlx.DB, pref *FoodPref) error {
	_, err := db.NamedExec(`INSERT INTO food_prefs (food_id, number_of_servings, serving_size)
                          VALUES (:food_id, :number_of_servings, :serving_size)
                          ON CONFLICT(food_id) DO UPDATE SET
                          number_of_servings = :number_of_servings,
                          serving_size = :serving_size`,
		pref)
	return err
}

// updateMealFoodPrefs updates the user's preferences for a given
// food in a meal.
func updateMealFoodPrefs(db *sqlx.DB, pref *MealFoodPref) error {
	_, err := db.NamedExec(`INSERT INTO meal_food_prefs (meal_id, food_id, number_of_servings, serving_size)
                          VALUES (:meal_id, :food_id, :number_of_servings, :serving_size)
                          ON CONFLICT(meal_id, food_id) DO UPDATE SET
                          number_of_servings = :number_of_servings,
                          serving_size = :serving_size`,
		pref)
	return err
}

// getOneFood retrieves the details for a given food.
// TODO:
// * For serving size and number of serving, first check `food_prefs`
// table for entries.
func getOneFood(db *sqlx.DB, foodID int) (*Food, error) {
	f := Food{}

	err := db.Get(&f, "SELECT * FROM foods WHERE food_id=?", foodID)
	if err != nil {
		return nil, err
	}

	err = db.Get(&f.PortionCals, "SELECT amount FROM food_nutrients WHERE food_id = ? AND nutrient_id IN (SELECT nutrient_id FROM nutrients WHERE nutrient_name = 'Energy' AND unit_name = 'KCAL' LIMIT 1)", foodID)
	if err != nil {
		return nil, err
	}

	f.Macros, err = getMacros(db, foodID)
	if err != nil {
		return nil, err
	}

	return &f, nil
}

// getMacros retrieves the macronutrients for a given food.
func getMacros(db *sqlx.DB, foodID int) (*Macros, error) {
	m := Macros{}

	nID, err := getNutrientId(db, "Protein")
	if err != nil {
		return nil, err
	}
	err = db.Get(&m.Protein, selectNutrientAmountSQL, foodID, nID)
	if err != nil {
		return nil, err
	}

	nID, err = getNutrientId(db, "Total lipid (fat)")
	if err != nil {
		return nil, err
	}
	err = db.Get(&m.Fat, selectNutrientAmountSQL, foodID, nID)
	if err != nil {
		return nil, err
	}

	nID, err = getNutrientId(db, "Carbohydrate, by difference")
	if err != nil {
		return nil, err
	}
	err = db.Get(&m.Carbs, selectNutrientAmountSQL, foodID, nID)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// getNutrientId retreives the `nutrient_id` for a given nutrient.
func getNutrientId(db *sqlx.DB, name string) (int, error) {
	var id int
	row := db.QueryRow(selectNutrientIdSQL, name)
	err := row.Scan(&id)
	if err != nil {
		log.Printf("Nutrient name \"%s\" does not exist.\n", name)
		return 0, err
	}

	return id, nil
}

// TODO: deleteFood
