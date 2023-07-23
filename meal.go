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
	FoodMacros       *FoodMacros
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

type FoodMacros struct {
	Protein float64
	Fat     float64
	Carbs   float64
}

const usage = "Usage: ./m [add|display]"

// TODO: REMOVE AFTER TESTING
func Run(db *sqlx.DB) {
	sf, _ := selectFood(db)
	f, err := getOneFood(db, sf.ID)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("ID:", f.ID)
	fmt.Println("Name:", f.Name)
	fmt.Println("Serving size:", f.ServingSize, f.ServingUnit)
	fmt.Println("PortionCals:", f.PortionCals)
	fmt.Println("Protein:", f.FoodMacros.Protein)
	fmt.Println("Carbs:", f.FoodMacros.Carbs)
	fmt.Println("Fats:", f.FoodMacros.Fat)

	/*
		meal := Meal{}
		meal.Name = promptMeal()
		addMeal(db, meal)

		m, _ := selectMeal(db)
		printMeal(m)
	*/
}

// promptDeleteMeal prompts a user to select a meal and removes the meal
// from the database.
func promptDeleteMeal(db *sqlx.DB) error {
	// Select a meal.
	m, err := selectMeal(db)
	if err != nil {
		return err
	}

	// Delete selected meal.
	deleteMeal(db, m.ID)
	return nil
}

// selectMeal prints the user's meals, prompts them to select a meal,
// and returns the selected meal.
func selectMeal(db *sqlx.DB) (Meal, error) {
	// Get all meals.
	meals, err := getAllMeals(db)
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
		filteredMeals, err := searchMeals(db, response)
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

// selectFood prompts user to enter a search term, prints the matched
// foods, prompts user to enter an index to select a food or another
// serach term for a different food. This repeats until user enters a
// valid index.
func selectFood(db *sqlx.DB) (Food, error) {
	// Get initial search term.
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter food name: ")
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	// Remove the newline character at the end of the string.
	response = strings.TrimSpace(response)

	// While user response is not an integer
	for {
		// Get filtered foods.
		filteredFoods, err := searchFoods(db, response)
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

// promptSelectResponse prompts and returns meal to select or a search term.
func promptSelectResponse(item string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter either the index of the %s to select or a search term: ", item)
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	// Remove the newline character at the end of the string
	response = strings.TrimSpace(response)
	return response
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
func searchMeals(db *sqlx.DB, response string) (*[]Meal, error) {
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
	err := db.Select(&meals, query, "%"+response+"%", response, response+"%", searchLimit)
	if err != nil {
		log.Printf("Search for meals failed: %v\n", err)
		return nil, err
	}

	return &meals, nil
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
	_, err := db.Exec(`INSERT INTO meals (meal_name) VALUES (?)`, meal.Name)
	if err != nil {
		log.Fatal("ERROR: ", err)
	}
}

// getAllMeals returns all the user's meals from the database.
func getAllMeals(db *sqlx.DB) ([]Meal, error) {
	m := []Meal{}
	err := db.Select(&m, "SELECT * FROM meals")
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

// displayAllMeals gets and prints all the user's meals.
func displayAllMeals(db *sqlx.DB) error {
	m, err := getAllMeals(db)
	if err != nil {
		return err
	}
	printMeals(m)
	return nil
}

// displayOneMeal gets and prints one user meal given the meal id.
func displayOneMeal(db *sqlx.DB, mealID int) error {
	m, err := getOneMeal(db, mealID)
	if err != nil {
		return err
	}
	printMeal(*m)
	return nil
}

// printMeals prints the fields of each meal in the given slice.
func printMeals(meals []Meal) {
	// TODO: ensure meals not empty

	// Print the meals.
	for _, meal := range meals {
		printMeal(meal)
	}
}

// printMeal prints the fields of the meal.
func printMeal(meal Meal) {
	// TODO: update print to include rest of the fields.
	fmt.Printf("ID: %d, Name: \"%s\"\n", meal.ID, meal.Name)
}

// TODO: this function should delete mealID from related tables. E.g.,
// meal_foods table.
func deleteMeal(db *sqlx.DB, mealID int) error {
	_, err := db.Exec("DELETE FROM meals WHERE id=?", mealID)
	return err
}

func incrementMealFrequency(db *sqlx.DB, mealID int) error {
	_, err := db.Exec(`UPDATE meals SET frequency = frequency + 1 WHERE id=?`, mealID)
	return err
}

// TODO: displayFood takes in a slice of meal id's to prints its contents.

// getFoodPref prompts user for food perferences, validates their
// response until they've entered a valid response, and returns the
// valid response.
//
// TODO:
// * This should be named updateFoodPref and should execute the sql code
// to make the update to the perferences.
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

	f.FoodMacros, err = getFoodMacros(db, foodID)
	if err != nil {
		return nil, err
	}

	return &f, nil
}

// getFoodMacros retrieves the macronutrients for a given food.
func getFoodMacros(db *sqlx.DB, foodID int) (*FoodMacros, error) {
	m := FoodMacros{}

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
