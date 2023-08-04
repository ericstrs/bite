package calories

import (
	"bufio"
	"errors"
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
	derivationIdPortion     = 71
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

type Queryer interface {
	QueryRow(query string, args ...interface{}) *sqlx.Row
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

	// Indicates if there is a serving size preference set for this food in
	// the meal (either in meal_food_prefs or food_prefs).
	HasPreference bool `db:"has_preference"`
}

type FoodPref struct {
	FoodID           int     `db:"food_id"`
	NumberOfServings float64 `db:"number_of_servings"`
	ServingSize      float64 `db:"serving_size"`
	ServingUnit      string  `db:"serving_unit"`
}

type MealFoodPref struct {
	FoodID           int     `db:"food_id"`
	MealID           int64   `db:"meal_id"`
	NumberOfServings float64 `db:"number_of_servings"`
	ServingSize      float64 `db:"serving_size"`
}

type FoodMacros struct {
	Protein float64
	Fat     float64
	Carbs   float64
}

// CreateAndAddFood creates a new food and adds it into the database.
func CreateAndAddFood(db *sqlx.DB) error {
	// Get food information.
	newFood, err := getNewFoodUserInput()
	if err != nil {
		return err
	}

	// Prompt user to input food nutrients information.
	newFoodMacros, cals, err := getFoodNutrientsUserInput(db)
	if err != nil {
		return fmt.Errorf("failed to get food nutrients user input: %w", err)
	}
	newFood.FoodMacros = newFoodMacros
	newFood.PortionCals = cals

	// Start a new transaction.
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	// Insert food into the foods table.
	newFood.ID, err = insertFood(tx, newFood)
	fmt.Println("newFood.ID:", newFood.ID)
	if err != nil {
		log.Println(err)
		return err
	}

	// Insert food nutrients into the food_nutrients table.
	err = insertFoodNutrientsIntoDB(tx, newFood)
	if err != nil {
		log.Printf("failed to insert food nutrients into database: %v", err)
		return fmt.Errorf("failed to insert food nutrients into database: %v", err)
	}

	fmt.Println("Added new food.")

	// Commit the transaction
	return tx.Commit()
}

// getNewFoodUserInput prompts user to enter new food information and
// returns Food struct.
func getNewFoodUserInput() (*Food, error) {
	newFood := &Food{}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter the food name: ")
	newFood.Name, _ = reader.ReadString('\n')

	// Remove newline character at the end
	newFood.Name = strings.TrimSuffix(newFood.Name, "\n")

	newFood.ServingSize = getServingSize()

	fmt.Printf("Enter serving unit: ")
	fmt.Scanln(&newFood.ServingUnit)

	fmt.Print("Enter the household serving: ")
	newFood.HouseholdServing, _ = reader.ReadString('\n')

	// Remove newline character at the end
	newFood.HouseholdServing = strings.TrimSuffix(newFood.HouseholdServing, "\n")

	return newFood, nil
}

// getFoodNutrientsUserInput retrieves the food nutrients from the user.
func getFoodNutrientsUserInput(db *sqlx.DB) (*FoodMacros, float64, error) {
	var foodMacros FoodMacros
	var cals float64
	nutrientNames := []string{"Protein", "Fat", "Carbs", "Energy"}

	for _, nutrientName := range nutrientNames {
		fmt.Printf("Enter the amount of %s per 100 serving units: ", nutrientName)
		var amount float64
		_, err := fmt.Scan(&amount)
		if err != nil {
			fmt.Println("Invalid input. Try again.")
			continue
		}
		if amount < 0 {
			fmt.Println("Nutrient amount cannot be negative.")
			continue
		}

		if nutrientName == "Energy" {
			cals = amount
			continue
		}

		switch nutrientName {
		case "Protein":
			foodMacros.Protein = amount
		case "Fat":
			foodMacros.Fat = amount
		case "Carbs":
			foodMacros.Carbs = amount
		default:
			fmt.Println("Invalid nutrient name.")
			continue
		}
	}

	return &foodMacros, cals, nil
}

// insertFood inserts a food into the database and returns the id of the newly inserted food.
func insertFood(tx *sqlx.Tx, food *Food) (int, error) {
	query := `INSERT INTO foods (food_name, serving_size, serving_unit, household_serving) VALUES (?, ?, ?, ?)`
	res, err := tx.Exec(query, food.Name, food.ServingSize, food.ServingUnit, food.HouseholdServing)
	if err != nil {
		return 0, fmt.Errorf("insertFood: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last inserted food ID: %w", err)
	}
	return int(id), nil
}

// insertNutrient inserts a nutrient into the food_nutrients table.
// nutrientID: The ID of the nutrient.
// foodID: The ID of the food.
// amount: The amount of the nutrient.
func insertNutrient(db *sqlx.DB, nutrientID int, foodID int, amount float64) error {
	// This is a constant for the food_nutrient_derivation id indicating
	// the nutrient value is for the portion size.
	const derivationID = 71

	// SQL statement for inserting a nutrient.
	insertNutrientSQL := `INSERT INTO food_nutrients (food_id, nutrient_id, amount, derivation_id) VALUES (?, ?, ?, ?)`

	// Insert nutrient.
	_, err := db.Exec(insertNutrientSQL, foodID, nutrientID, amount, derivationID)
	if err != nil {
		fmt.Printf("Failed to insert nutrient with ID %d: %v\n", nutrientID, err)
		return err
	}

	return nil
}

// insertFoodNutrientsIntoDB inserts the nutrients of a food into the food_nutrients table.
func insertFoodNutrientsIntoDB(tx *sqlx.Tx, food *Food) error {
	// Nutrients and corresponding amounts.
	nutrients := map[string]float64{
		"Protein":                     food.FoodMacros.Protein,
		"Total lipid (fat)":           food.FoodMacros.Fat,
		"Carbohydrate, by difference": food.FoodMacros.Carbs,
		"Energy":                      food.PortionCals,
	}

	insertFoodNutrientsSQL := `
    INSERT INTO food_nutrients (food_id, nutrient_id, amount, derivation_id)
    VALUES (:food_id, :nutrient_id, :amount, :derivation_id)
	`

	// Insert each nutrient into the food_nutrients table.
	for nutrientName, amount := range nutrients {
		nutrientID, err := getNutrientId(tx, nutrientName)
		if err != nil {
			continue // Skip this nutrient if there was an error retrieving the ID.
		}

		// Create a map with named parameters
		params := map[string]interface{}{
			"food_id":       food.ID,
			"nutrient_id":   nutrientID,
			"amount":        amount,
			"derivation_id": derivationIdPortion,
		}

		_, err = tx.NamedExec(insertFoodNutrientsSQL, params)
		if err != nil {
			return err
		}
	}

	return nil
}

// SelectAndDeleteFood prompts user to select food to delete and removes
// the food from the database.
func SelectAndDeleteFood(db *sqlx.DB) error {
	// Get food to delete.
	food, err := selectFood(db)
	if err != nil {
		if errors.Is(err, ErrDone) {
			fmt.Println("No food selected.")
			return nil // Not really an "error" situation
		}
		return err
	}

	// Delete food.
	err = deleteFood(db, food.ID)
	if err != nil {
		return err
	}
	fmt.Println("Deleted food.")
	return nil
}

// deleteFood deletes a food from the database.
func deleteFood(db *sqlx.DB, foodID int) error {
	// Start a new transaction
	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
		return err
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	// Execute the delete statement
	_, err = tx.Exec(`
      DELETE FROM meal_food_prefs
      WHERE food_id = $1
      `, foodID)

	// If there was an error executing the query, return the error
	if err != nil {
		log.Printf("Couldn't delete entry from meal_food_prefs: %v\n", err)
		return err
	}

	_, err = tx.Exec(`
			DELETE FROM food_prefs
			WHERE food_id = $1
			`, foodID)

	// If there was an error executing the query, return the error
	if err != nil {
		log.Printf("Couldn't delete entry from meal_food_prefs: %v\n", err)
		return err
	}

	_, err = tx.Exec(`
			DELETE FROM meal_foods
			WHERE food_id = $1
			`, foodID)

	// If there was an error executing the query, return the error
	if err != nil {
		log.Printf("Couldn't delete entry from meal_foods: %v\n", err)
		return err
	}

	_, err = tx.Exec(`
			DELETE FROM food_nutrients
			WHERE food_id = $1
			`, foodID)

	// If there was an error executing the query, return the error
	if err != nil {
		log.Printf("Couldn't delete entry from meal_foods: %v\n", err)
		return err
	}

	_, err = tx.Exec(`
			DELETE FROM daily_foods
			WHERE food_id = $1
			`, foodID)

	// If there was an error executing the query, return the error
	if err != nil {
		log.Printf("Couldn't delete entry from daily_foods: %v\n", err)
		return err
	}

	_, err = tx.Exec(`
			DELETE FROM food_nutrients
			WHERE food_id = $1
			`, foodID)

	// If there was an error executing the query, return the error
	if err != nil {
		log.Printf("Couldn't delete entry from food_nutrients: %v\n", err)
		return err
	}

	_, err = tx.Exec(`
			DELETE FROM food_nutrients
			WHERE food_id = $1
			`, foodID)

	// If there was an error executing the query, return the error
	if err != nil {
		log.Printf("Couldn't delete entry from food_nutrients: %v\n", err)
		return err
	}

	_, err = tx.Exec(`
			DELETE FROM foods
			WHERE food_id = $1
			`, foodID)

	// If there was an error executing the query, return the error
	if err != nil {
		log.Printf("Couldn't delete entry from foods: %v\n", err)
		return err
	}

	// If everything went fine, commit the transaction
	return tx.Commit()
}

// CreateAndAddMeal creates a new meal and adds it into the database.
func CreateAndAddMeal(db *sqlx.DB) error {
	// Get meal information.
	mealName := promptMealName()

	// Insert the meal into the meals table.
	mealID, err := insertMeal(db, mealName)
	if err != nil {
		log.Println(err)
		return err
	}

	// Now prompt the user to enter the foods that make up the meal.
	for {
		// Select a food.
		food, err := selectFood(db)
		if err != nil {
			if errors.Is(err, ErrDone) {
				break // If the user entered "done", break the loop.
			}
			log.Println(err)
			continue // If there was a different error, ask again.
		}

		// Insert food into meal food table.
		err = insertMealFood(db, int(mealID), food.ID)
		if err != nil {
			return err
		}

		// Get any existing preferences for the selected food.
		f, err := getMealFoodWithPref(db, food.ID, mealID)
		if err != nil {
			log.Println(err)
			return err
		}

		// Display any existing preferences for the selected food.
		printMealFood(f)

		var s string
		fmt.Printf("Do you want to change these values? (y/n): ")
		fmt.Scan(&s)

		// If the user decides to change existing food preferences,
		if strings.ToLower(s) == "y" {
			// Get updated food preferences.
			mf := getMealFoodPrefUserInput(food.ID, mealID)
			// Make database entry for meal food preferences.
			err := updateMealFoodPrefs(db, mf)
			if err != nil {
				return err
			}
		}
	}

	fmt.Println("Added meal.")
	return nil
}

// SelectAndDeleteMeal selects a meal deletes a meal from the
// database.
func SelectAndDeleteMeal(db *sqlx.DB) error {
	// Select meal to delete.
	m, err := selectMeal(db)
	if err != nil {
		return err
	}

	// Remove meal from the database.
	err = deleteMeal(db, m.ID)
	if err != nil {
		return err
	}

	fmt.Println("Deleted food.")
	return nil
}

// deleteMeal deletes a meal from the database.
func deleteMeal(db *sqlx.DB, mealID int) error {
	// Start a new transaction
	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
		return err
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	_, err = tx.Exec(`
      DELETE FROM meal_food_prefs
      WHERE meal_id = $1
      `, mealID)
	// If there was an error executing the query, return the error
	if err != nil {
		log.Printf("Couldn't delete entry from meal_food_prefs: %v\n", err)
		return err
	}

	_, err = tx.Exec(`
      DELETE FROM meal_foods
      WHERE meal_id = $1
      `, mealID)
	// If there was an error executing the query, return the error
	if err != nil {
		log.Printf("Couldn't delete entry from meal_foods: %v\n", err)
		return err
	}

	// Set any `meal_id` in the daily_foods table for any entries that
	// were apart of this meal to NULL.
	_, err = tx.Exec(`UPDATE daily_foods SET meal_id = NULL WHERE meal_id = ?`, mealID)
	// If there was an error executing the query, return the error
	if err != nil {
		log.Printf("Failed to update daily_foods: %v\n", err)
		return err
	}

	_, err = tx.Exec(`
      DELETE FROM daily_meals
      WHERE meal_id = $1
      `, mealID)
	// If there was an error executing the query, return the error
	if err != nil {
		log.Printf("Couldn't delete entry from daily_meals: %v\n", err)
		return err
	}

	_, err = tx.Exec(`
      DELETE FROM meals
      WHERE meal_id = $1
      `, mealID)
	// If there was an error executing the query, return the error
	if err != nil {
		log.Printf("Couldn't delete entry from meals: %v\n", err)
		return err
	}

	return tx.Commit()
}

// insertMeal inserts a meal into the database and returns the id of the
// newly inserted meal.
func insertMeal(db *sqlx.DB, mealName string) (int64, error) {
	query := `INSERT INTO meals (meal_name) VALUES ($1)`
	res, err := db.Exec(query, mealName)
	if err != nil {
		return 0, fmt.Errorf("insertMeal: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last inserted ID: %w", err)
	}
	return id, nil
}

// insertMealFood inserts a food that is part of a meal into the
// database.
func insertMealFood(db *sqlx.DB, mealID, foodID int) error {
	// Start a new transaction
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute the insert query
	_, err = tx.Exec(`
        INSERT INTO meal_foods (meal_id, food_id)
        VALUES ($1, $2)
    `, mealID, foodID)

	if err != nil {
		return err
	}

	return tx.Commit()
}

// updateMealFoodPrefs inserts or updates the user's preferences for a
// given food that is part of a meal.
func updateMealFoodPrefs(db *sqlx.DB, pref *MealFoodPref) error {
	// Start a new transaction
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	// Execute the update statement
	_, err = tx.NamedExec(`
			INSERT INTO meal_food_prefs (meal_id, food_id, number_of_servings, serving_size)
      VALUES (:meal_id, :food_id, :number_of_servings, :serving_size)
      ON CONFLICT(meal_id, food_id) DO UPDATE SET
      number_of_servings = :number_of_servings,
      serving_size = :serving_size`, pref)

	// If there was an error executing the query, return the error
	if err != nil {
		return err
	}

	// If everything went fine, commit the transaction
	return tx.Commit()
}

// GetUserInputAddMealFood prompts the user for existing meal and food
// to add to the meal and then inserts the new meal food into the
// database.
func GetUserInputAddMealFood(db *sqlx.DB) error {
	// Select existing meal.
	meal, err := selectMeal(db)
	if err != nil {
		return err
	}

	// Select a food.
	food, err := selectFood(db)
	if err != nil {
		if errors.Is(err, ErrDone) {
			return err // If the user entered "done", return early.
		}
		log.Println(err)
	}

	// Insert food into meal food table.
	err = insertMealFood(db, meal.ID, food.ID)
	if err != nil {
		return err
	}

	// Get any existing preferences for the selected food.
	f, err := getMealFoodWithPref(db, food.ID, int64(meal.ID))
	if err != nil {
		log.Println(err)
		return err
	}

	// Display any existing preferences for the selected food.
	printMealFood(f)

	var s string
	fmt.Printf("Do you want to change these values? (y/n): ")
	fmt.Scan(&s)

	// If the user decides to change existing food preferences,
	if strings.ToLower(s) == "y" {
		// Get updated food preferences.
		mf := getMealFoodPrefUserInput(food.ID, int64(meal.ID))
		// Make database entry for meal food preferences.
		err := updateMealFoodPrefs(db, mf)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Successfully added food %d to meal %d\n", food.ID, meal.ID)
	return nil
}

// SelectAndDeleteMealFood prompts user to select a meal and a food to
// remove.
func SelectAndDeleteFoodMealFood(db *sqlx.DB) error {
	// Select meal.
	meal, err := selectMeal(db)
	if err != nil {
		return err
	}

	// Select a food.
	food, err := selectFood(db)
	if err != nil {
		if errors.Is(err, ErrDone) {
			return err // If the user entered "done", return early.
		}
		log.Println(err)
	}

	// Delete meal food.
	err = deleteMealFood(db, meal.ID, food.ID)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully removed food %d to meal %d\n", food.ID, meal.ID)
	return nil
}

// deleteMealFood removes a food that is part of a meal from the
// database.
func deleteMealFood(db *sqlx.DB, mealID, foodID int) error {
	// Start a new transaction
	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
		return err
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	_, err = tx.Exec(`
      DELETE FROM meal_food_prefs
      WHERE meal_id = $1 AND food_id = $2
      `, mealID, foodID)
	// If there was an error executing the query, return the error
	if err != nil {
		log.Printf("Couldn't delete entry from meal_food_prefs: %v\n", err)
		return err
	}

	_, err = tx.Exec(`
      DELETE FROM meal_foods
      WHERE meal_id = $1 AND food_id = $2
      `, mealID, foodID)
	// If there was an error executing the query, return the error
	if err != nil {
		log.Printf("Couldn't delete entry from meal_foods: %v\n", err)
		return err
	}

	// Set any `meal_id` in the daily_foods table for any entries for this
	// meal food to NULL.
	_, err = tx.Exec(`UPDATE daily_foods SET meal_id = NULL WHERE meal_id = $1 AND food_id =$2`, mealID, foodID)
	// If there was an error executing the query, return the error
	if err != nil {
		log.Printf("Failed to update daily_foods: %v\n", err)
		return err
	}

	return tx.Commit()
}

// promptMealName prompts and returns name of meal.
func promptMealName() (m string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter the name of your new meal: ")
	m, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	return strings.TrimSpace(m)
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
		log.Println("Failed to get protein nutrient ID.")
		return nil, err
	}
	err = db.Get(&m.Protein, selectNutrientAmountSQL, foodID, nID)
	if err != nil {
		log.Println("Failed to get protein nutrient for food.")
		return nil, err
	}

	nID, err = getNutrientId(db, "Total lipid (fat)")
	if err != nil {
		log.Println("Failed to get fat nutrient ID.")
		return nil, err
	}
	err = db.Get(&m.Fat, selectNutrientAmountSQL, foodID, nID)
	if err != nil {
		log.Println("Failed to get fat nutrient for food.")
		return nil, err
	}

	nID, err = getNutrientId(db, "Carbohydrate, by difference")
	if err != nil {
		log.Println("Failed to get carbohydrate nutrient ID.")
		return nil, err
	}
	err = db.Get(&m.Carbs, selectNutrientAmountSQL, foodID, nID)
	if err != nil {
		log.Println("Failed to get carbohydrate nutrient for food.")
		return nil, err
	}

	return &m, nil
}

// getNutrientId retrieves the `nutrient_id` for a given nutrient.
func getNutrientId(ext sqlx.Ext, name string) (int, error) {
	var id int
	err := sqlx.Get(ext, &id, selectNutrientIdSQL, name)
	if err != nil {
		log.Printf("Nutrient name \"%s\" does not exist.\n", name)
		return 0, err
	}

	return id, nil
}
