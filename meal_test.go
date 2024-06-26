package bite

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
)

func ExampleInsertFood() {
	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Start a new transaction.
	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
		return
	}

	// Create the foods table
	tx.MustExec(` CREATE TABLE IF NOT EXISTS foods (
  food_id INTEGER PRIMARY KEY,
  food_name TEXT NOT NULL,
  serving_size REAL NOT NULL,
  serving_unit TEXT NOT NULL,
  household_serving TEXT NOT NULL
  )`)

	food := Food{
		ID:               1,
		Name:             "Chicken Breast",
		ServingUnit:      "g",
		ServingSize:      100,
		HouseholdServing: "1 piece",
	}

	// Insert food into table.
	if _, err = InsertFood(tx, food); err != nil {
		fmt.Println(err)
		return
	}

	// Verify the food was inserted.
	var newFood Food
	err = tx.Get(&newFood, `SELECT * FROM foods WHERE food_id = 1`)
	if err != nil {
		fmt.Printf("ERROR: couldn't get foods: %v", err)
	}

	// Commit the transaction.
	if err := tx.Commit(); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("ID:", newFood.ID)
	fmt.Println("Name:", newFood.Name)
	fmt.Println("Serving Unit:", newFood.ServingUnit)
	fmt.Println("Serving Size:", newFood.ServingSize)
	fmt.Println("Household Serving:", newFood.HouseholdServing)
	fmt.Println(err)

	// Output:
	// ID: 1
	// Name: Chicken Breast
	// Serving Unit: g
	// Serving Size: 100
	// Household Serving: 1 piece
	// <nil>
}

func ExampleUpdateFoodTable() {
	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Start a new transaction
	tx, err := db.Beginx()
	if err != nil {
		return
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	// Create the food_nutrients table
	tx.MustExec(`
			CREATE TABLE IF NOT EXISTS foods (
				food_id INTEGER PRIMARY KEY,
				food_name TEXT NOT NULL,
				serving_size REAL NOT NULL,
				serving_unit TEXT NOT NULL,
				household_serving TEXT NOT NULL,
				brand_name TEXT DEFAULT '',
				cost REAL DEFAULT 0
			);
	`)

	tx.MustExec(`INSERT INTO foods (food_id, food_name, serving_size, serving_unit, household_serving) VALUES
  (1, 'Chicken Breast', 100, 'g', '1/2 piece')
  `)

	updatedFood := &Food{
		ID:               1,
		Name:             "Chicken Breast",
		ServingSize:      50,
		ServingUnit:      "g",
		HouseholdServing: "1/2 piece",
		BrandName:        "Great Value",
		Price:            15,
	}

	if err := UpdateFoodTable(tx, updatedFood); err != nil {
		fmt.Println(err)
		return
	}

	var selectedFood Food
	tx.Get(&selectedFood, `SELECT * FROM foods WHERE food_id = 1`)

	fmt.Println(selectedFood.Name)
	fmt.Println(selectedFood.ServingSize)
	fmt.Println(selectedFood.ServingUnit)
	fmt.Println(selectedFood.HouseholdServing)
	fmt.Println(selectedFood.BrandName)
	fmt.Println(selectedFood.Price)

	// Output:
	// Chicken Breast
	// 50
	// g
	// 1/2 piece
	// Great Value
	// 15
}

func ExampleDeleteFood() {
	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Start a new transaction
	tx, err := db.Beginx()
	if err != nil {
		return
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	// Create the food_nutrients table
	tx.MustExec(`
			-- foods contains static information about foods.
			CREATE TABLE IF NOT EXISTS foods (
				food_id INTEGER PRIMARY KEY,
				food_name TEXT NOT NULL,
				serving_size REAL NOT NULL,
				serving_unit TEXT NOT NULL,
				household_serving TEXT NOT NULL
			);

			-- user_foods contains the user's food consumption
			-- logs.
			CREATE TABLE IF NOT EXISTS daily_foods (
				id INTEGER PRIMARY KEY,
				food_id INTEGER REFERENCES foods(food_id) NOT NULL,
				meal_id INTEGER REFERENCES meals(meal_id),
				date DATE NOT NULL,
				serving_size REAL NOT NULL,
				number_of_servings REAL DEFAULT 1 NOT NULL
			);

			-- meal_foods relates meals to the foods the contain.
			CREATE TABLE IF NOT EXISTS meal_foods (
				meal_id INTEGER REFERENCES meals(meal_id),
				food_id INTEGER REFERENCES foods(food_id),
				number_of_servings REAL DEFAULT 1 NOT NULL
			);

			-- nutrients stores the nurtients that a food can be comprised of.
			CREATE TABLE IF NOT EXISTS nutrients (
				nutrient_id INTEGER PRIMARY KEY,
				nutrient_name TEXT NOT NULL,
				unit_name TEXT NOT NULL
			);

			-- food_nutrient_derivation stores the procedure indicating how a food
			-- nutrient value was obtained.
			CREATE TABLE IF NOT EXISTS food_nutrient_derivation (
				id INT PRIMARY KEY,
				code VARCHAR(255) NOT NULL,
				description TEXT NOT NULL
			);

			CREATE TABLE IF NOT EXISTS food_nutrients (
				id INTEGER PRIMARY KEY,
				food_id INTEGER NOT NULL,
				nutrient_id INTEGER NOT NULL,
				amount REAL NOT NULL,
				derivation_id REAL NOT NULL,
				FOREIGN KEY (food_id) REFERENCES foods(food_id),
				FOREIGN KEY (nutrient_id) REFERENCES nutrients(nutrients_id),
				FOREIGN KEY (derivation_id) REFERENCES food_nutrient_derivation(id)
			);

			CREATE TABLE IF NOT EXISTS food_prefs (
				food_id INTEGER PRIMARY KEY,
				serving_size REAL,
				number_of_servings REAL DEFAULT 1 NOT NULL,
				FOREIGN KEY(food_id) REFERENCES foods(food_id)
			);

			CREATE TABLE IF NOT EXISTS meal_food_prefs (
				meal_id INTEGER,
				food_id INTEGER,
				serving_size REAL,
				number_of_servings REAL DEFAULT 1 NOT NULL,
				PRIMARY KEY(meal_id, food_id),
				FOREIGN KEY(food_id) REFERENCES foods(food_id),
				FOREIGN KEY(meal_id) REFERENCES meals(meal_id)
			);
  `)

	// Insert food
	tx.MustExec(`INSERT INTO foods (food_id, food_name, serving_size, serving_unit, household_serving) VALUES
  (1, 'Chicken Breast', 100, 'g', '1/2 piece')
	`)

	// Then, insert a nutrient
	tx.MustExec(`INSERT INTO nutrients (nutrient_id, nutrient_name, unit_name) VALUES
	(1003, 'Protein', 'g')
	`)

	// Insert into daily_foods
	tx.MustExec(`INSERT INTO daily_foods (food_id, meal_id, date, serving_size, number_of_servings) VALUES
	(1, 1, '2023-07-09', 100, 1)
	`)

	// Insert into meal_foods
	tx.MustExec(`INSERT INTO meal_foods (meal_id, food_id, number_of_servings) VALUES
	(1, 1, 1)
	`)

	// Insert into food_nutrients
	tx.MustExec(`INSERT INTO food_nutrients (food_id, nutrient_id, amount, derivation_id) VALUES
	(1, 1003, 50, 71)
	`)

	// Insert into food_prefs
	tx.MustExec(`INSERT INTO food_prefs (food_id, serving_size, number_of_servings) VALUES
	(1, 100, 1)
	`)

	// Insert into meal_food_prefs
	tx.MustExec(`INSERT INTO meal_food_prefs (meal_id, food_id, serving_size, number_of_servings) VALUES
	(1, 1, 100, 1)
	`)

	if err := DeleteFood(tx, 1); err != nil {
		fmt.Printf("ERROR: %v", err)
	}

	tx.Commit()

	// Verify food was deleted
	tables := []string{"daily_foods", "meal_foods", "food_nutrients", "food_prefs", "meal_food_prefs"}
	foodID := 1

	for _, table := range tables {
		var id int
		err = db.Get(&id, fmt.Sprintf("SELECT food_id FROM %s WHERE food_id = ?", table), foodID)
		if err != nil {
			if err == sql.ErrNoRows {
				fmt.Printf("Food with ID %d was successfully deleted from table %s.\n", foodID, table)
			} else {
				fmt.Printf("An error occurred while checking table %s: %s\n", table, err.Error())
			}
		} else {
			fmt.Printf("Food with ID %d was not deleted from table %s.\n", foodID, table)
		}
	}

	// Output:
	// Food with ID 1 was successfully deleted from table daily_foods.
	// Food with ID 1 was successfully deleted from table meal_foods.
	// Food with ID 1 was successfully deleted from table food_nutrients.
	// Food with ID 1 was successfully deleted from table food_prefs.
	// Food with ID 1 was successfully deleted from table meal_food_prefs.
}

func ExampleInsertMeal() {
	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Start a new transaction
	tx, err := db.Beginx()
	if err != nil {
		return
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	// Create meals table.
	tx.MustExec(`
		CREATE TABLE IF NOT EXISTS meals (
				meal_id INTEGER PRIMARY KEY,
				meal_name TEXT NOT NULL
		);
	`)

	id, err := InsertMeal(tx, "Cereal")
	if err != nil {
		fmt.Println(err)
	}

	tx.Commit()

	var mealName string
	err = db.Get(&mealName, `SELECT meal_name FROM meals WHERE meal_id = $1`, id)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(mealName)
	fmt.Println(err)

	// Output:
	// Cereal
	// <nil>
}

func ExampleInsertMealFood() {
	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Start a new transaction
	tx, err := db.Beginx()
	if err != nil {
		return
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	// Create meal foods table.
	tx.MustExec(`
		CREATE TABLE IF NOT EXISTS meals (
				meal_id INTEGER PRIMARY KEY,
				meal_name TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS foods (
			food_id INTEGER PRIMARY KEY,
			food_name TEXT NOT NULL,
			serving_size REAL NOT NULL,
			serving_unit TEXT NOT NULL,
			household_serving TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS meal_foods (
			meal_id INTEGER REFERENCES meals(meal_id),
			food_id INTEGER REFERENCES foods(food_id),
			PRIMARY KEY (meal_id, food_id)
		);
  `)

	_, err = tx.Exec(`INSERT INTO meals VALUES (1, 'Cereal')`)
	if err != nil {
		log.Printf("Failed to insert data into meal table: %v\n", err)
		return
	}

	_, err = tx.Exec(`INSERT INTO foods (food_id, food_name, serving_size, serving_unit, household_serving) VALUES
  (1, 'Milk', 240, 'g', '1 cup')
	`)
	if err != nil {
		log.Printf("Failed to insert data into foods table: %v\n", err)
		return
	}

	if err := InsertMealFood(tx, 1, 1); err != nil {
		fmt.Println(err)
	}

	tx.Commit()

	var mealID float64
	err = db.Get(&mealID, `SELECT meal_id FROM meal_foods WHERE meal_id = 1 AND food_id = 1`, 1, 1)
	if err != nil {
		fmt.Println(err)
	}

	var foodID float64
	err = db.Get(&foodID, `SELECT food_id FROM meal_foods WHERE meal_id = 1 AND food_id = 1`, 1, 1)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(mealID)
	fmt.Println(foodID)
	fmt.Println(err)

	// Output:
	// 1
	// 1
	// <nil>
}

func ExampleUpdateMealFoodPrefs() {
	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Start a new transaction
	tx, err := db.Beginx()
	if err != nil {
		return
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	tx.MustExec(`
		CREATE TABLE IF NOT EXISTS foods (
			food_id INTEGER PRIMARY KEY,
			food_name TEXT NOT NULL,
			serving_size REAL NOT NULL,
			serving_unit TEXT NOT NULL,
			household_serving TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS meals (
				meal_id INTEGER PRIMARY KEY,
				meal_name TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS meal_food_prefs (
			meal_id INTEGER,
			food_id INTEGER,
			serving_size REAL,
			number_of_servings REAL DEFAULT 1 NOT NULL,
			PRIMARY KEY(meal_id, food_id),
			FOREIGN KEY(food_id) REFERENCES foods(food_id),
			FOREIGN KEY(meal_id) REFERENCES meals(meal_id)
		);
	`)

	_, err = tx.Exec(`INSERT INTO meals VALUES (1, 'Cereal')`)
	if err != nil {
		log.Printf("Failed to insert data into meal table: %v\n", err)
		return
	}

	_, err = tx.Exec(`INSERT INTO foods (food_id, food_name, serving_size, serving_unit, household_serving) VALUES
  (1, 'Milk', 240, 'g', '1 cup')
	`)
	if err != nil {
		log.Printf("Failed to insert data into foods table: %v\n", err)
		return
	}

	pref := &MealFoodPref{}
	pref.FoodID = 1
	pref.MealID = 1
	pref.ServingSize = 300
	pref.NumberOfServings = 1

	if err := UpdateMealFoodPrefs(tx, *pref); err != nil {
		log.Printf("Failed to updated meal food prefs: %v\n", err)
	}

	tx.Commit()

	// Ensure serving size food preference was updated.
	var servingSize float64
	err = db.Get(&servingSize, `SELECT serving_size FROM meal_food_prefs WHERE meal_id = 1 AND food_id = 1`)
	if err != nil {
		log.Printf("Failed to get updated serving size: %v\n", servingSize)
		return
	}
	fmt.Println(servingSize)
	fmt.Println(err)

	// Output:
	// 300
	// <nil>
}

func ExampleDeleteMeal() {
	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		fmt.Printf("ERROR: couldn't connect to test database: %v", err)
	}
	defer db.Close()

	// Start a new transaction
	tx, err := db.Beginx()
	if err != nil {
		return
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	// Create the food_nutrients table
	tx.MustExec(`
			-- foods contains static information about foods.
			CREATE TABLE IF NOT EXISTS foods (
				food_id INTEGER PRIMARY KEY,
				food_name TEXT NOT NULL,
				serving_size REAL NOT NULL,
				serving_unit TEXT NOT NULL,
				household_serving TEXT NOT NULL
			);

			CREATE TABLE IF NOT EXISTS meals (
 		   meal_id INTEGER PRIMARY KEY,
   		 meal_name TEXT NOT NULL
			);

			-- user_foods contains the user's food consumption
			-- logs.
			CREATE TABLE IF NOT EXISTS daily_foods (
				id INTEGER PRIMARY KEY,
				food_id INTEGER REFERENCES foods(food_id) NOT NULL,
				meal_id INTEGER REFERENCES meals(meal_id),
				date DATE NOT NULL,
				serving_size REAL NOT NULL,
				number_of_servings REAL DEFAULT 1 NOT NULL
			);

			CREATE TABLE IF NOT EXISTS daily_meals (
  			id INTEGER PRIMARY KEY,
  			meal_id INTEGER REFERENCES meals(meal_id),
  			date DATE NOT NULL
			);

			-- meal_foods relates meals to the foods the contain.
			CREATE TABLE IF NOT EXISTS meal_foods (
				meal_id INTEGER REFERENCES meals(meal_id),
				food_id INTEGER REFERENCES foods(food_id),
				number_of_servings REAL DEFAULT 1 NOT NULL
			);

			-- nutrients stores the nurtients that a food can be comprised of.
			CREATE TABLE IF NOT EXISTS nutrients (
				nutrient_id INTEGER PRIMARY KEY,
				nutrient_name TEXT NOT NULL,
				unit_name TEXT NOT NULL
			);

			-- food_nutrient_derivation stores the procedure indicating how a food
			-- nutrient value was obtained.
			CREATE TABLE IF NOT EXISTS food_nutrient_derivation (
				id INT PRIMARY KEY,
				code VARCHAR(255) NOT NULL,
				description TEXT NOT NULL
			);

			CREATE TABLE IF NOT EXISTS food_nutrients (
				id INTEGER PRIMARY KEY,
				food_id INTEGER NOT NULL,
				nutrient_id INTEGER NOT NULL,
				amount REAL NOT NULL,
				derivation_id REAL NOT NULL,
				FOREIGN KEY (food_id) REFERENCES foods(food_id),
				FOREIGN KEY (nutrient_id) REFERENCES nutrients(nutrients_id),
				FOREIGN KEY (derivation_id) REFERENCES food_nutrient_derivation(id)
			);

			CREATE TABLE IF NOT EXISTS meal_food_prefs (
				meal_id INTEGER,
				food_id INTEGER,
				serving_size REAL,
				number_of_servings REAL DEFAULT 1 NOT NULL,
				PRIMARY KEY(meal_id, food_id),
				FOREIGN KEY(food_id) REFERENCES foods(food_id),
				FOREIGN KEY(meal_id) REFERENCES meals(meal_id)
			);
  `)
	// Insert food
	tx.MustExec(`INSERT INTO foods (food_id, food_name, serving_size, serving_unit, household_serving) VALUES
  (1, 'Chicken Breast', 100, 'g', '1/2 piece')
  `)

	// Then, insert a nutrient
	tx.MustExec(`INSERT INTO nutrients (nutrient_id, nutrient_name, unit_name) VALUES
  (1003, 'Protein', 'g')
  `)

	// Insert into meals
	tx.MustExec(`INSERT INTO meals (meal_name) VALUES
	('Chicken burrito')
	`)

	// Insert into daily_meals
	tx.MustExec(`INSERT INTO daily_meals (meal_id, date) VALUES
  (1, '2023-07-09')
  `)

	// Insert into daily_foods
	tx.MustExec(`INSERT INTO daily_foods (food_id, meal_id, date, serving_size, number_of_servings) VALUES
  (1, 1, '2023-07-09', 100, 1)
  `)

	// Insert into meal_foods
	tx.MustExec(`INSERT INTO meal_foods (meal_id, food_id, number_of_servings) VALUES
  (1, 1, 1)
  `)

	// Insert into food_nutrients
	tx.MustExec(`INSERT INTO food_nutrients (food_id, nutrient_id, amount, derivation_id) VALUES
  (1, 1003, 50, 71)
  `)

	// Insert into meal_food_prefs
	tx.MustExec(`INSERT INTO meal_food_prefs (meal_id, food_id, serving_size, number_of_servings) VALUES
  (1, 1, 100, 1)
  `)

	if err := DeleteMeal(tx, 1); err != nil {
		fmt.Printf("Failed to delete meal: %v\n", err)
		return
	}

	tx.Commit()

	// Verify food was deleted
	tables := []string{"daily_foods", "meal_foods", "meals", "meal_food_prefs"}
	mealID := 1

	for _, table := range tables {
		var df DailyFood
		err = db.Get(&df, fmt.Sprintf("SELECT meal_id FROM %s WHERE meal_id = ?", table), mealID)
		if err != nil {
			if err == sql.ErrNoRows {
				fmt.Printf("Meal with ID %d was successfully deleted from table %s.\n", mealID, table)
			} else {
				fmt.Printf("An error occurred while checking table %s: %s\n", table, err.Error())
			}
		} else {
			fmt.Printf("Meal with ID %d was not deleted from table %s.\n", mealID, table)
		}
	}

	// Output:
	// Meal with ID 1 was successfully deleted from table daily_foods.
	// Meal with ID 1 was successfully deleted from table meal_foods.
	// Meal with ID 1 was successfully deleted from table meals.
	// Meal with ID 1 was successfully deleted from table meal_food_prefs.
}

func ExampleDeleteMealFood() {
	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Start a new transaction
	tx, err := db.Beginx()
	if err != nil {
		return
	}
	// If anything goes wrong, rollback the transaction
	defer tx.Rollback()

	// Create the food_nutrients table
	tx.MustExec(`
      -- foods contains static information about foods.
      CREATE TABLE IF NOT EXISTS foods (
        food_id INTEGER PRIMARY KEY,
        food_name TEXT NOT NULL,
        serving_size REAL NOT NULL,
        serving_unit TEXT NOT NULL,
        household_serving TEXT NOT NULL
      );

      CREATE TABLE IF NOT EXISTS meals (
       meal_id INTEGER PRIMARY KEY,
       meal_name TEXT NOT NULL
      );

      -- user_foods contains the user's food consumption
      -- logs.
      CREATE TABLE IF NOT EXISTS daily_foods (
        id INTEGER PRIMARY KEY,
        food_id INTEGER REFERENCES foods(food_id) NOT NULL,
        meal_id INTEGER REFERENCES meals(meal_id),
        date DATE NOT NULL,
        serving_size REAL NOT NULL,
        number_of_servings REAL DEFAULT 1 NOT NULL
      );

			CREATE TABLE IF NOT EXISTS meal_foods (
        meal_id INTEGER REFERENCES meals(meal_id),
        food_id INTEGER REFERENCES foods(food_id)
      );

			CREATE TABLE IF NOT EXISTS meal_food_prefs (
        meal_id INTEGER,
        food_id INTEGER,
        serving_size REAL,
        number_of_servings REAL DEFAULT 1 NOT NULL,
        PRIMARY KEY(meal_id, food_id),
        FOREIGN KEY(food_id) REFERENCES foods(food_id),
        FOREIGN KEY(meal_id) REFERENCES meals(meal_id)
      );
	`)
	// Insert into meals
	tx.MustExec(`INSERT INTO meals (meal_name) VALUES
	('Chicken burrito')
	`)

	// Insert food
	tx.MustExec(`INSERT INTO foods (food_id, food_name, serving_size, serving_unit, household_serving) VALUES
  (1, 'Chicken Breast', 100, 'g', '1/2 piece')
  `)

	// Insert into meal_foods
	tx.MustExec(`INSERT INTO meal_foods (meal_id, food_id) VALUES
  (1, 1)
  `)

	// Insert into meal_food_prefs
	tx.MustExec(`INSERT INTO meal_food_prefs (meal_id, food_id, serving_size, number_of_servings) VALUES
  (1, 1, 1.5, 1)
  `)

	if err := DeleteMealFood(tx, 1, 1); err != nil {
		log.Println(err)
		return
	}

	tx.Commit()

	// Verify meal food was deleted
	tables := []string{"meal_foods", "meal_food_prefs"}
	mealID := 1
	foodID := 1

	for _, table := range tables {
		var id float64
		err = db.Get(&id, fmt.Sprintf("SELECT meal_id FROM %s WHERE meal_id = ? AND food_id = ?", table), mealID, foodID)
		if err != nil {
			if err == sql.ErrNoRows {
				fmt.Printf("Meal with ID %d and Food with ID %d was successfully deleted from table %s.\n", mealID, foodID, table)
			} else {
				fmt.Printf("An error occurred while checking table %s: %s\n", table, err.Error())
			}
		} else {
			fmt.Printf("Meal with ID %d and Food with ID %d was not deleted from table %s.\n", mealID, foodID, table)
		}
	}

	// Output:
	// Meal with ID 1 and Food with ID 1 was successfully deleted from table meal_foods.
	// Meal with ID 1 and Food with ID 1 was successfully deleted from table meal_food_prefs.
}
