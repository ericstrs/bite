package calories

import (
	"bufio"
	"fmt"
	"log"
	"os"
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

// MealFood extends Food with additional fields to represent a food
// as part of a meal.
type MealFood struct {
	Food
	// NumberOfServings represents the amount of the food that is part of the meal.
	// The value is derived based on the following order:
	// 1. From the meal_food_prefs table if a specific preference for this food
	//    in the context of the given meal exists.
	// 2. From the food_prefs table if a general preference for this food exists.
	// 3. From the foods table which contains default values for each food.
	NumberOfServings float64 `db:"number_of_servings"`

	// ServingSize represents the size of each serving of the food that is part of the meal.
	// The value is derived in the same way and the same precedence as NumberOfServings.
	ServingSize float64 `db:"serving_size"`
}

type FoodPref struct {
	FoodId           int     `db:"food_id"`
	NumberOfServings float64 `db:"number_of_servings"`
	ServingSize      float64 `db:"serving_size"`
	ServingUnit      string  `db:"serving_unit"`
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

func promptDeleteResponse() (r string) {
	// Prompt for user input
	fmt.Printf("Enter the index of the meal to delete or a search term: ")
	_, err := fmt.Scanln(&r)
	if err != nil {
		log.Fatal(err)
	}
	return r
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

// TODO: displayFood takes in a slice of meal id's to prints its contents.

// getServingSize prompts user for serving size, validates their
// response until they've entered a valid response, and returns the
// valid response.
func getServingSize() float64 {
	fmt.Print("Enter new serving size: ")

	var servingSize float64
	var err error
	for {
		// Prompt for serving size.
		servingSize, err = promptServingSize()
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
	if 0 >= servingSize || servingSize > 1000 {
		return fmt.Errorf("Serving size must be between 0 and 1000")
	}
	return nil
}

// getNumServings prompts user for number of servings, validates their
// response until they've entered a valid response, and returns the
// valid response.
func getNumServings() float64 {
	fmt.Print("Enter new number of servings: ")
	var numOfServings float64
	var err error
	for {
		// Prompt user for number of servings.
		numOfServings, err = promptNumServings()
		if err != nil {
			fmt.Println("Invalid number of servings: Number of servings must be a number. Please try again.")
			continue
		}

		// Validate user response.
		err = validateNumServings(numOfServings)
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
	if 0 >= numServings || numServings > 100 {
		return fmt.Errorf("Number of servings must be between 0 and 100.")
	}
	return nil
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
// Nutrients
// Nutrients are for portion size (100 serving unit)
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
		log.Println("NO PROTEIN")
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
