package bite

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

const (
	derivationIdPortion = 71
)

type Meal struct {
	ID        int    `db:"meal_id"`
	Name      string `db:"meal_name"`
	Frequency int
	Foods     []MealFood
	Cals      float64 // Total meal calories
	Protein   float64 // Total meal protein
	Carbs     float64 // Total meal carbs
	Fats      float64 // Total meal fats
}

type Food struct {
	ID               int     `db:"food_id"`
	Name             string  `db:"food_name"`
	ServingUnit      string  `db:"serving_unit"`
	ServingSize      float64 `db:"serving_size"`
	NumberOfServings float64 `db:"number_of_servings"`
	HouseholdServing string  `db:"household_serving"`
	Calories         float64
	FoodMacros       *FoodMacros
	// Indicates if there is a serving size preference set for this food in
	// the meal (in food_prefs).
	BrandName string  `db:"brand_name"`
	Price     float64 `db:"cost"`
}

// MealFood extends Food with additional fields to represent a food
// as part of a meal.
type MealFood struct {
	MealID int
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
}

type FoodPref struct {
	FoodID           int     `db:"food_id"`
	NumberOfServings float64 `db:"number_of_servings"`
	ServingSize      float64 `db:"serving_size"`
	ServingUnit      string  `db:"serving_unit"`
	HouseholdServing string  `db:"household_serving"`
}

type MealFoodPref struct {
	FoodID           int     `db:"food_id"`
	MealID           int64   `db:"meal_id"`
	NumberOfServings float64 `db:"number_of_servings"`
	ServingSize      float64 `db:"serving_size"`
}

type FoodMacros struct {
	Protein float64 `db:"protein"`
	Fat     float64 `db:"fat"`
	Carbs   float64 `db:"carbs"`
}

// CreateAddFood creates a new food and adds it into the database.
func CreateAddFood(db *sqlx.DB) error {
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
	newFood.Calories = cals

	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert food into the foods table.
	newFood.ID, err = InsertFood(tx, *newFood)
	if err != nil {
		return err
	}

	// Insert food nutrients into the food_nutrients table.
	if err = InsertNutrients(db, tx, *newFood); err != nil {
		return fmt.Errorf("failed to insert food nutrients into database: %v", err)
	}

	fmt.Println("Added new food.")

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

	fmt.Printf("Enter the household serving: ")
	newFood.HouseholdServing, _ = reader.ReadString('\n')
	// Remove newline character at the end
	newFood.HouseholdServing = strings.TrimSuffix(newFood.HouseholdServing, "\n")

	fmt.Printf("Enter the food's brand name [Press <Enter> to skip]: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	newFood.BrandName = input

	newFood.Price = getFoodPriceUserInput()

	return newFood, nil
}

// getFoodPriceUserInput prompts user for price of a given food, validates user
// response, and returns the valid food price.
func getFoodPriceUserInput() float64 {
	reader := bufio.NewReader(os.Stdin)

	var floatValue float64
	var err error
	for {
		fmt.Printf("Enter food price per 100 serving units [Press <Enter> to skip]: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSuffix(input, "\n")

		if input == "" {
			return 0
		}

		floatValue, err = strconv.ParseFloat(input, 64)
		if err != nil || floatValue < 0 {
			fmt.Println("Value must be a number greater than 0. Please try again.")
			continue
		}
		return floatValue
	}
}

// getFoodNutrientsUserInput retrieves the food nutrients from the user.
func getFoodNutrientsUserInput(db *sqlx.DB) (*FoodMacros, float64, error) {
	var foodMacros FoodMacros
	nutrientNames := []string{"Protein", "Fat", "Carbs"}

	for _, nutrientName := range nutrientNames {
		fmt.Printf("Enter the amount of %s per 100 serving units: ", nutrientName)
		var amount float64
		_, err := fmt.Scan(&amount)
		if err != nil || amount < 0 {
			fmt.Println("Invalid input. Try again.")
			continue
		}
		if amount < 0 {
			fmt.Println("Nutrient amount cannot be negative.")
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

	cals := CalculateCalories(foodMacros.Protein, foodMacros.Carbs, foodMacros.Fat)

	return &foodMacros, cals, nil
}

// CalculateCalories calculates the calories of a food given
// macronutrient amounts.
func CalculateCalories(protein, carbs, fats float64) float64 {
	return (protein * calsInProtein) + (carbs * calsInCarbs) + (fats * calsInFats)
}

// InsertFood inserts a food into the database and returns the id of the newly inserted food.
func InsertFood(tx *sqlx.Tx, food Food) (int, error) {
	const query = `
	INSERT INTO foods (food_name, serving_size, serving_unit, household_serving)
	VALUES ($1, $2, $3, $4)
	`
	res, err := tx.Exec(query, food.Name, food.ServingSize, food.ServingUnit, food.HouseholdServing)
	if err != nil {
		return 0, fmt.Errorf("InsertFood: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last inserted food ID: %w", err)
	}
	return int(id), nil
}

// insertNutrient inserts a nutrient into the food_nutrients table.
func insertNutrient(db *sqlx.DB, nutrientID int, foodID int, amount float64) error {
	const (
		// SQL statement for inserting a nutrient.
		nutrientSQL = `INSERT INTO food_nutrients (food_id, nutrient_id, amount, derivation_id)
		VALUES (?, ?, ?, ?)
	`
		// This is a constant for the food_nutrient_derivation id indicating
		// the nutrient value is for the portion size.
		derivationID = 71
	)

	_, err := db.Exec(nutrientSQL, foodID, nutrientID, amount, derivationID)
	if err != nil {
		fmt.Printf("Failed to insert nutrient with ID %d: %v\n", nutrientID, err)
		return err
	}

	return nil
}

// InsertNutrients inserts the nutrients of a food into the food_nutrients table.
func InsertNutrients(db *sqlx.DB, tx *sqlx.Tx, food Food) error {
	// Nutrients and corresponding amounts.
	nutrients := map[string]float64{
		"Protein":                     food.FoodMacros.Protein,
		"Total lipid (fat)":           food.FoodMacros.Fat,
		"Carbohydrate, by difference": food.FoodMacros.Carbs,
		"Energy":                      food.Calories,
	}

	insertFoodNutrientsSQL := `
    INSERT INTO food_nutrients (food_id, nutrient_id, amount, derivation_id)
    VALUES (:food_id, :nutrient_id, :amount, :derivation_id)
	`

	// Insert each nutrient into the food_nutrients table.
	for nutrientName, amount := range nutrients {
		nutrientID, err := getNutrientId(db, nutrientName)
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

		if _, err = tx.NamedExec(insertFoodNutrientsSQL, params); err != nil {
			return err
		}
	}

	return nil
}

// UpdateFood prompts user for new food information and makes the update
// to the database.
func UpdateFood(db *sqlx.DB) error {
	food, err := selectFood(db)
	if err != nil {
		if errors.Is(err, ErrDone) {
			fmt.Println("No food selected.")
			return nil // Not really an "error" situation
		}
		return err
	}

	// Get new food information
	updateFoodUserInput(&food)

	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
		return err
	}
	defer tx.Rollback()

	// Make update to foods table
	if err := UpdateFoodTable(tx, &food); err != nil {
		return err
	}

	// Get existing food macros
	food.FoodMacros, err = getFoodMacros(db, food.ID)
	if err != nil {
		return err
	}

	// Get new food nutrient information
	updateFoodNutrientsUserInput(&food)

	// Make update to food nutrients table
	err = UpdateFoodNutrients(db, tx, &food)
	if err != nil {
		return err
	}

	fmt.Printf("updated food %q.\n", food.Name)

	return tx.Commit()
}

// updateFoodUserInput prompts the user to update information for an existing food.
func updateFoodUserInput(existingFood *Food) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Current food name: %s\n", existingFood.Name)
	fmt.Printf("Enter new food name [Press <Enter> to keep]: ")
	newName, _ := reader.ReadString('\n')
	newName = strings.TrimSpace(newName)
	if newName != "" {
		existingFood.Name = newName
	}

	existingFood.ServingSize = updateServingSizeUserInput(existingFood.ServingSize)

	fmt.Printf("Current serving unit: %s\n", existingFood.ServingUnit)
	fmt.Printf("Enter new serving unit [Press <Enter> to keep]: ")
	newServingUnit, _ := reader.ReadString('\n')
	newServingUnit = strings.TrimSpace(newServingUnit)
	if newServingUnit != "" {
		existingFood.ServingUnit = newServingUnit
	}

	fmt.Printf("Current household serving: %s\n", existingFood.HouseholdServing)
	fmt.Printf("Enter new household serving [Press <Enter> to keep]: ")
	newHouseholdServing, _ := reader.ReadString('\n')
	newHouseholdServing = strings.TrimSpace(newHouseholdServing)
	if newHouseholdServing != "" {
		existingFood.HouseholdServing = newHouseholdServing
	}

	fmt.Printf("Current brand name: %s\n", existingFood.BrandName)
	fmt.Printf("Enter new brand name [Press <Enter> to keep]: ")
	newBrandName, _ := reader.ReadString('\n')
	newBrandName = strings.TrimSpace(newBrandName)
	if newBrandName != "" {
		existingFood.BrandName = newBrandName
	}

	existingFood.Price = updateFoodPriceUserInput(existingFood.Price)
}

// updateServingSizeUserInput entered prints existing food serving size and prompts user
// to enter a new one.
func updateServingSizeUserInput(existingServingSize float64) float64 {
	var newServingSize string
	fmt.Printf("Current serving size: %.2f\n", existingServingSize)
	for {
		fmt.Printf("Enter new serving size [Press <Enter> to keep]: ")
		fmt.Scanln(&newServingSize)

		// User pressed <Enter>
		if newServingSize == "" {
			return existingServingSize
		}

		newServingSizeFloat, err := strconv.ParseFloat(newServingSize, 64)
		if err != nil || newServingSizeFloat < 0 {
			fmt.Println("Invalid float value entered. Please try again.")
			continue
		}
		return newServingSizeFloat
	}
}

// updateFoodPriceUserInput prints current price for food prompts user
// for price of a given food, validates user response, and returns the
// valid food price.
func updateFoodPriceUserInput(existingFoodPrice float64) float64 {
	var newFoodPrice string
	fmt.Printf("Current food price per 100 servings units: $%.2f\n", existingFoodPrice)
	for {
		fmt.Printf("Enter food price per 100 serving units [Press <Enter> to keep]: ")
		fmt.Scanln(&newFoodPrice)

		// User pressed <Enter>
		if newFoodPrice == "" {
			return existingFoodPrice
		}

		newFoodPriceFloat, err := strconv.ParseFloat(newFoodPrice, 64)
		if err != nil || newFoodPriceFloat < 0 {
			fmt.Println("Value must be a number greater than 0. Please try again.")
			continue
		}
		return newFoodPriceFloat
	}
}

// UpdateFoodTable updates one food from the foods table
func UpdateFoodTable(tx *sqlx.Tx, food *Food) error {
	const query = `
		UPDATE foods SET
		food_name = $1, serving_size = $2, serving_unit = $3,
		household_serving = $4, brand_name = $5, cost = $6
		WHERE food_id = $7
	`
	_, err := tx.Exec(query, food.Name, food.ServingSize, food.ServingUnit,
		food.HouseholdServing, food.BrandName, food.Price, food.ID)
	if err != nil {
		return fmt.Errorf("Failed to update food: %v", err)
	}

	return nil
}

// updateFoodNutrientsUserInput prints existing nutrient information,
// prompts user for new nutrient information, and returns information.
//
// Assumption:
// * f.FoodMacros is not empty
func updateFoodNutrientsUserInput(f *Food) error {
	nutrientNames := []string{"Protein", "Fat", "Carbs"}
	var newAmount string
	var newAmountFloat float64
	var err error

OuterLoop:
	for _, nutrientName := range nutrientNames {
		// Print existing nutrient information
		fmt.Printf("Current %s amount per 100 serving units: ", strings.ToLower(nutrientName))
		switch nutrientName {
		case "Protein":
			fmt.Printf("%.0f\n", f.FoodMacros.Protein)
		case "Fat":
			fmt.Printf("%.0f\n", f.FoodMacros.Fat)
		case "Carbs":
			fmt.Printf("%.0f\n", f.FoodMacros.Carbs)
		default:
			fmt.Println("\nInvalid nutrient name:", nutrientName)
			continue
		}

		for {
			fmt.Printf("Enter new amount per 100 serving units [Press <Enter> to keep]: ")
			fmt.Scanln(&newAmount)

			// User pressed <Enter>
			if newAmount == "" {
				continue OuterLoop
			}

			newAmountFloat, err = strconv.ParseFloat(newAmount, 64)
			if err != nil || newAmountFloat < 0 {
				fmt.Println("Nutrient amount be a number greater than 0. Please try again.")
				continue
			}
		}

		switch nutrientName {
		case "Protein":
			f.FoodMacros.Protein = newAmountFloat
		case "Fat":
			f.FoodMacros.Fat = newAmountFloat
		case "Carbs":
			f.FoodMacros.Carbs = newAmountFloat
		default:
			fmt.Println("Invalid nutrient name:", nutrientName)
			continue
		}
	}

	f.Calories = CalculateCalories(f.FoodMacros.Protein, f.FoodMacros.Carbs, f.FoodMacros.Fat)

	return nil
}

// UpdateFoodNutrients updates the food nutrients for a given food.
func UpdateFoodNutrients(db *sqlx.DB, tx *sqlx.Tx, food *Food) error {
	// Nutrients and corresponding amounts.
	nutrients := map[string]float64{
		"Protein":                     food.FoodMacros.Protein,
		"Total lipid (fat)":           food.FoodMacros.Fat,
		"Carbohydrate, by difference": food.FoodMacros.Carbs,
		"Energy":                      food.Calories,
	}

	updateFoodNutrientsSQL := `
    UPDATE food_nutrients SET
		amount = :amount, derivation_id = :derivation_id
		WHERE nutrient_id = :nutrient_id AND food_id = :food_id
  `

	// Insert each nutrient into the food_nutrients table.
	for nutrientName, amount := range nutrients {
		nutrientID, err := getNutrientId(db, nutrientName)
		if err != nil {
			log.Println("ERROR: ", err)
			continue // Skip this nutrient if there was an error retrieving the ID.
		}

		// Create a map with named parameters
		params := map[string]interface{}{
			"food_id":       food.ID,
			"nutrient_id":   nutrientID,
			"amount":        amount,
			"derivation_id": derivationIdPortion,
		}

		_, err = tx.NamedExec(updateFoodNutrientsSQL, params)
		if err != nil {
			log.Println("Failed to update food nutrients:", err)
			return err
		}
	}

	return nil
}

// SelectDeleteFood prompts user to select food to delete and removes
// the food from the database.
func SelectDeleteFood(db *sqlx.DB) error {
	food, err := selectFood(db)
	if err != nil {
		if errors.Is(err, ErrDone) {
			fmt.Println("No food selected.")
			return nil // Not really an "error" situation
		}
		return err
	}

	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
		return err
	}
	defer tx.Rollback()

	if err := DeleteFood(tx, food.ID); err != nil {
		return err
	}
	fmt.Println("Deleted food.")

	return tx.Commit()
}

// DeleteFood deletes a food from the database.
func DeleteFood(tx *sqlx.Tx, foodID int) error {
	// Execute the delete statement
	_, err := tx.Exec(`
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

	return nil
}

// CreateAddMeal creates a new meal and adds it into the database.
func CreateAddMeal(db *sqlx.DB) error {
	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
	}
	defer tx.Rollback()

	// Get meal information.
	mealName := promptMealName()

	// Insert the meal into the meals table.
	mealID, err := InsertMeal(tx, mealName)
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
		if err := InsertMealFood(tx, int(mealID), food.ID); err != nil {
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
			mf := getMealFoodPrefUserInput(food.ID, mealID, f.ServingSize, f.NumberOfServings)
			// Make database entry for meal food preferences.
			if err := UpdateMealFoodPrefs(tx, *mf); err != nil {
				return err
			}
		}
	}

	fmt.Println("Successfully created meal.")

	// Commit the transaction
	return tx.Commit()
}

// UpdateMeal updates an existing meal.
func UpdateMeal(tx *sqlx.Tx, m Meal) error {
	const updateSQL = `
		UPDATE meals
    SET meal_name = $1
    WHERE meal_id = $2
	`
	_, err := tx.Exec(updateSQL, m.Name, m.ID)
	return err
}

// SelectDeleteMeal selects as meal deletes it from the database.
func SelectDeleteMeal(db *sqlx.DB) error {
	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
		return err
	}
	defer tx.Rollback()

	m, err := selectMeal(db)
	if err != nil {
		return err
	}

	// Store meal name before deleting.
	mealName := m.Name

	// Remove meal from the database.
	if err := DeleteMeal(tx, m.ID); err != nil {
		return err
	}

	fmt.Printf("Successfully deleted %s meal.\n", mealName)
	return tx.Commit()
}

// DeleteMeal deletes a meal from the database.
func DeleteMeal(tx *sqlx.Tx, mealID int) error {
	_, err := tx.Exec(`
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

	return nil
}

// InsertMeal inserts a meal into the database and returns the id of the
// newly inserted meal.
func InsertMeal(tx *sqlx.Tx, mealName string) (int64, error) {
	query := `INSERT INTO meals (meal_name) VALUES ($1)`
	res, err := tx.Exec(query, mealName)
	if err != nil {
		return 0, fmt.Errorf("InsertMeal: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last inserted ID: %w", err)
	}
	return id, nil
}

// InsertMealFood inserts a food that is part of a meal into the
// database.
func InsertMealFood(tx *sqlx.Tx, mealID, foodID int) error {
	// Execute the insert query
	_, err := tx.Exec(`
        INSERT INTO meal_foods (meal_id, food_id)
        VALUES ($1, $2)
    `, mealID, foodID)

	if err != nil {
		return err
	}

	return nil
}

// UpdateMealFoodPrefs inserts or updates the user's preferences for a
// given food that is part of a meal.
func UpdateMealFoodPrefs(tx *sqlx.Tx, pref MealFoodPref) error {
	// Execute the update statement
	_, err := tx.NamedExec(`
			INSERT INTO meal_food_prefs (meal_id, food_id, number_of_servings, serving_size)
      VALUES (:meal_id, :food_id, :number_of_servings, :serving_size)
      ON CONFLICT(meal_id, food_id) DO UPDATE SET
      number_of_servings = :number_of_servings,
      serving_size = :serving_size`, pref)
	return err
}

// PromptAddMealFood prompts for existing meal and food to add to the
// meal and then inserts the new meal food into the database.
func PromptAddMealFood(db *sqlx.DB) error {
	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
		return err
	}
	defer tx.Rollback()

	meal, err := selectMeal(db)
	if err != nil {
		return err
	}

	food, err := selectFood(db)
	if err != nil {
		if errors.Is(err, ErrDone) {
			return err // If the user entered "done", return early.
		}
		log.Println(err)
	}

	// Insert food into meal food table.
	if err := InsertMealFood(tx, meal.ID, food.ID); err != nil {
		return err
	}

	// Get any existing preferences for the selected food.
	mealFood, err := getMealFoodWithPref(db, food.ID, int64(meal.ID))
	if err != nil {
		log.Println(err)
		return err
	}

	// Display any existing preferences for the selected food.
	printMealFood(mealFood)

	var s string
	fmt.Printf("Do you want to change these values? (y/n): ")
	fmt.Scan(&s)

	// If the user decides to change existing food preferences,
	if strings.ToLower(s) == "y" {
		// Get updated food preferences.
		mf := getMealFoodPrefUserInput(food.ID, int64(meal.ID), mealFood.ServingSize, mealFood.NumberOfServings)
		// Make database entry for meal food preferences.
		if err := UpdateMealFoodPrefs(tx, *mf); err != nil {
			return err
		}
	}

	fmt.Printf("Successfully added %s to %s meal\n", mealFood.Name, meal.Name)
	return tx.Commit()
}

// SelectDeleteMealFood prompts user to select a meal and then select
// one of its food to remove.
func SelectDeleteFoodMealFood(db *sqlx.DB) error {
	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
		return err
	}
	defer tx.Rollback()

	meal, err := selectMeal(db)
	if err != nil {
		return err
	}

	// Get the foods that make up the meal.
	mealFoods, err := GetMealFoodsWithPref(db, meal.ID)
	if err != nil {
		log.Println(err)
		return err
	}

	// Print the foods that make up the meal and their preferences.
	printMealDetails(mealFoods)

	// Let user select food to delete.
	var idx int
	for {
		// Get user response.
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Enter index of food to remove: ")
		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("SelectDeleteFoodMealFood: %v\n", err)
		}
		// Remove the newline character at the end of the string
		response = strings.TrimSpace(response)

		idx, err = strconv.Atoi(response)

		// If user enters an invalid integer,
		if 1 > idx || idx > len(mealFoods) {
			fmt.Println("Number must be between 0 and number of foods. Please try again.")
			continue
		}
		break
	}

	// Using selected food ID, remove it from the meal.
	if err := DeleteMealFood(tx, meal.ID, mealFoods[idx-1].Food.ID); err != nil {
		return err
	}

	fmt.Printf("Successfully removed %s from %s meal.\n", mealFoods[idx-1].Food.Name, meal.Name)
	return tx.Commit()
}

// DeleteMealFood removes a food that is part of a meal from the
// database.
func DeleteMealFood(tx *sqlx.Tx, mealID, foodID int) error {
	_, err := tx.Exec(`
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
	if err != nil {
		log.Printf("Couldn't delete entry from meal_foods: %v\n", err)
		return err
	}

	// Set any `meal_id` in the daily_foods table for any entries for this
	// meal food to NULL.
	_, err = tx.Exec(`UPDATE daily_foods SET meal_id = NULL WHERE meal_id = $1 AND food_id =$2`, mealID, foodID)
	if err != nil {
		log.Printf("Failed to update daily_foods: %v\n", err)
		return err
	}

	return nil
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
func getAllMeals(tx *sqlx.Tx) ([]Meal, error) {
	m := []Meal{}
	err := tx.Select(&m, "SELECT * FROM meals")
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

/*
// getOneFood retrieves the details for a given food.
// Nutrients
// Nutrients are for portion size (100 serving unit)
func getOneFood(tx *sqlx.Tx, foodID int) (*Food, error) {
	f := Food{}

	err := tx.Get(&f, "SELECT * FROM foods WHERE food_id=?", foodID)
	if err != nil {
		return nil, err
	}

	err = tx.Get(&f.Calories, "SELECT amount FROM food_nutrients WHERE food_id = ? AND nutrient_id IN (SELECT nutrient_id FROM nutrients WHERE nutrient_name = 'Energy' AND unit_name = 'KCAL' LIMIT 1)", foodID)
	if err != nil {
		return nil, err
	}

	f.FoodMacros, err = getFoodMacros(tx, foodID)
	if err != nil {
		return nil, err
	}

	return &f, nil
}
*/

// getFoodMacros retrieves the macronutrients for a given food.
func getFoodMacros(db *sqlx.DB, foodID int) (*FoodMacros, error) {
	const nutrientSQL = `
		SELECT COALESCE (
		  (SELECT amount
			 FROM food_nutrients
			 WHERE food_id = $1 AND nutrient_id = $2
			 ), 0) as amount
    LIMIT 1
		`
	stmt, err := db.Preparex(nutrientSQL)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	m := FoodMacros{}

	nID, err := getNutrientId(db, `Protein`)
	if err != nil {
		return nil, fmt.Errorf("couldn't get nutrient id: %v", err)
	}
	if err := stmt.Get(&m.Protein, foodID, nID); err != nil {
		return nil, fmt.Errorf("couldn't get protein: %v", err)
	}

	nID, err = getNutrientId(db, `Total lipid (fat)`)
	if err != nil {
		return nil, fmt.Errorf("couldn't get nutrient id: %v", err)
	}
	if err := stmt.Get(&m.Fat, foodID, nID); err != nil {
		return nil, fmt.Errorf("couldn't get FAT: %v", err)
	}

	nID, err = getNutrientId(db, `Carbohydrate, by difference`)
	if err != nil {
		return nil, fmt.Errorf("couldn't get nutrient id: %v", err)
	}
	if err := stmt.Get(&m.Carbs, foodID, nID); err != nil {
		return nil, fmt.Errorf("couldn't get carbs: %v", err)
	}

	return &m, nil
}

// getNutrientId retrieves the `nutrient_id` for a given nutrient.
func getNutrientId(db *sqlx.DB, name string) (int, error) {
	const selectNutrientIdSQL = `
	SELECT nutrient_id
	FROM nutrients
	WHERE nutrient_name = $1
	`

	var id int
	err := db.Get(&id, selectNutrientIdSQL, name)
	if err != nil {
		return 0, fmt.Errorf("nutrient name %q does not exist: %v", name, err)
	}

	return id, nil
}
