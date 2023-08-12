package calories

import (
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

func ExampleGetAllEntries() {
	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}

	defer db.Close()

	// Create the food_nutrient_derivation table
	db.MustExec(`CREATE TABLE food_nutrient_derivation (
  id INT PRIMARY KEY,
  code VARCHAR(255) NOT NULL,
  description TEXT NOT NULL
	)`)

	// Insert a derivation code and description
	db.MustExec(`INSERT INTO food_nutrient_derivation (code, description) VALUES (71, "portion size")`)

	// Create the foods table
	db.MustExec(` CREATE TABLE IF NOT EXISTS foods (
  food_id INTEGER PRIMARY KEY,
  food_name TEXT NOT NULL,
  serving_size REAL NOT NULL,
  serving_unit TEXT NOT NULL,
  household_serving TEXT NOT NULL
	)`)

	// Insert foods
	db.MustExec(`INSERT INTO foods (food_id, food_name, serving_size, serving_unit, household_serving) VALUES
	(1, 'Chicken Breast', 100, 'g', '1/2 piece'),
	(2, 'Broccoli', 156, 'g', '1 cup'),
	(3, 'Brown Rice', 100, 'g', '1/2 cup cooked'),
	(4, 'Pizza', 124, 'g', '1 slice'),
	(5, 'Taco', 170, 'g', '1 taco')
	`)

	// Create the nutrients table
	db.MustExec(`CREATE TABLE IF NOT EXISTS nutrients (
  nutrient_id INTEGER PRIMARY KEY,
  nutrient_name TEXT NOT NULL,
  unit_name TEXT NOT NULL
	)`)

	// Insert nutrients
	db.MustExec(`INSERT INTO nutrients (nutrient_id, nutrient_name, unit_name) VALUES
	(1003, 'Protein', 'G'),
	(1004, 'Total lipid (fat)', 'G'),
	(1005, 'Carbohydrate, by difference', 'G'),
	(1008, 'Energy, KCAL', 'G')
	`)

	// Create the food_nutrients table
	db.MustExec(`CREATE TABLE food_nutrients (
  id INTEGER PRIMARY KEY,
  food_id INTEGER NOT NULL,
  nutrient_id INTEGER NOT NULL,
  amount REAL NOT NULL,
  derivation_id REAL NOT NULL,
  FOREIGN KEY (food_id) REFERENCES foods(food_id),
  FOREIGN KEY (nutrient_id) REFERENCES nutrients(nutrients_id),
  FOREIGN KEY (derivation_id) REFERENCES food_nutrient_derivation(id)
	)`)

	// Insert food_nutrients
	db.MustExec(`INSERT INTO food_nutrients (food_id, nutrient_id, amount, derivation_id) VALUES
	(1, 1003, 31, 71),
	(1, 1004, 3.6, 71),
	(1, 1005, 0, 71),
	(1, 1008, 165, 71),
	(2, 1003, 2.8, 71),
	(2, 1004, 0.4, 71),
	(2, 1005, 7, 71),
	(2, 1008, 34, 71),
	(3, 1003, 2.73, 71),
	(3, 1004, 0.96, 71),
	(3, 1005, 25.5, 71),
	(3, 1008, 122, 71),
	(4, 1003, 11, 71),
	(4, 1004, 10, 71),
	(4, 1005, 33, 71),
	(4, 1008, 266, 71),
	(5, 1003, 12, 71),
	(5, 1004, 12, 71),
	(5, 1005, 15, 71),
	(5, 1008, 216, 71)
	`)

	// Create the daily_foods table
	db.MustExec(`CREATE TABLE daily_foods (
  id INTEGER PRIMARY KEY,
	food_id INTEGER REFERENCES foods(food_id) NOT NULL,
  meal_ID INTEGER REFERENCES meals(meal_id),
  date DATE NOT NULL,
	time TIME NOT NULL,
  number_of_servings REAL DEFAULT 1 NOT NULL,
	calories REAL NOT NULL,
  protein REAL NOT NULL,
  fat REAL NOT NULL,
  carbs REAL NOT NULL
	)`)

	// Note: 5th day user did not log any foods.
	db.MustExec(`INSERT INTO daily_foods (food_id, date, time, number_of_servings, calories, protein, fat, carbs) VALUES
		(1, '2023-01-01', '00:00:00', 1, 165, 31, 3.6, 0),
		(2, '2023-01-01', '00:00:00', 1, 34, 2.8, 0.4, 7),
		(2, '2023-01-02', '00:00:00', 1, 34, 2.8, 0.4, 7),
		(4, '2023-01-02', '00:00:00', 1, 266, 11, 10, 33),
		(5, '2023-01-03', '00:00:00', 1, 216, 12, 12, 15),
		(1, '2023-01-03', '00:00:00', 1, 165, 31, 3.6, 0),
		(3, '2023-01-04', '00:00:00', 1, 122, 2.73, 0.96, 25.5),
		(4, '2023-01-04', '00:00:00', 1, 266, 11, 10, 33)
	`)

	// Create the daily_weights table
	db.MustExec(`CREATE TABLE daily_weights (
  id INTEGER PRIMARY KEY,
  date DATE NOT NULL,
	time TIME NOT NULL,
  weight REAL NOT NULL
	)`)

	db.MustExec(`INSERT INTO daily_weights (date, time, weight) VALUES
	('2023-01-01', "00:00:00", 180),
	('2023-01-02', "00:00:00", 181),
	('2023-01-03', "00:00:00", 182),
	('2023-01-04', "00:00:00", 183),
	('2023-01-05', "00:00:00", 184)
	`)

	db.MustExec(`CREATE TABLE IF NOT EXISTS daily_meals (
  id INTEGER PRIMARY KEY,
  meal_id INTEGER REFERENCES meals(meal_id),
  date DATE NOT NULL,
	time TIME NOT NULL
	)`)

	db.MustExec(`CREATE TABLE IF NOT EXISTS food_prefs (
  food_id INTEGER PRIMARY KEY,
  serving_size REAL,
  number_of_servings REAL DEFAULT 1 NOT NULL,
  FOREIGN KEY(food_id) REFERENCES foods(food_id)
	)`)

	db.MustExec(`CREATE TABLE IF NOT EXISTS meal_food_prefs (
  meal_id INTEGER,
  food_id INTEGER,
  serving_size REAL,
  number_of_servings REAL DEFAULT 1 NOT NULL,
  PRIMARY KEY(meal_id, food_id),
  FOREIGN KEY(food_id) REFERENCES foods(food_id),
  FOREIGN KEY(meal_id) REFERENCES meals(meal_id)
	)`)

	// This tests to ensure that entering food and meal food preference
	// only affect future food logging. That is, the inserts below should
	// not change the output values.
	_, err = db.Exec(`
				INSERT INTO food_prefs VALUES (1, 160, 1);
		  	INSERT INTO meal_food_prefs VALUES (1, 1, 180, 2);
			`)

	// Get all entries
	entries, err := GetAllEntries(db)
	if err != nil {
		panic(err)
	}

	for _, entry := range *entries {
		fmt.Println("Date: ", entry.Date.Format(dateFormat))
		fmt.Println("Weight: ", entry.UserWeight)
		fmt.Println("Calories: ", entry.Calories)
	}

	// Output:
	// Date:  2023-01-01
	// Weight:  180
	// Calories:  199
	// Date:  2023-01-02
	// Weight:  181
	// Calories:  300
	// Date:  2023-01-03
	// Weight:  182
	// Calories:  381
	// Date:  2023-01-04
	// Weight:  183
	// Calories:  388
}

func ExampleAddWeightEntry() {
	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	db.MustExec(`CREATE TABLE IF NOT EXISTS daily_weights (
  id INTEGER PRIMARY KEY,
  date DATE NOT NULL,
	time TIME NOT NULL,
  weight REAL NOT NULL
)`)

	testWeight := 220.2
	date := time.Now()

	err = addWeightEntry(db, date, testWeight)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Verify the weight was logged correctly
	var weight float64
	err = db.Get(&weight, `SELECT weight FROM daily_weights WHERE date = ?`, date.Format(dateFormat))

	fmt.Println(weight)
	fmt.Println(err)

	// Output:
	// Added weight entry.
	// 220.2
	// <nil>
}

func ExampleAddWeightEntry_exists() {
	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	db.MustExec(`CREATE TABLE IF NOT EXISTS daily_weights (
  id INTEGER PRIMARY KEY,
  date DATE NOT NULL,
  weight REAL NOT NULL,
	time TIME NOT NULL
)`)

	testWeight := 220.2
	date := time.Now()

	// Insert a weight for date.
	db.Exec(`INSERT INTO daily_weights (date, time, weight) VALUES ($1, $2, $3)`, date.Format(dateFormat), date.Format(dateFormatTime), testWeight)

	// Attempt to insert another weight for same date.
	err = addWeightEntry(db, date, testWeight)
	fmt.Println(err)

	// Output:
	// Weight for this date has already been logged.
}

func ExampleCheckWeightExists() {
	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	db.MustExec(`CREATE TABLE IF NOT EXISTS daily_weights (
  id INTEGER PRIMARY KEY,
  date DATE NOT NULL,
  weight REAL NOT NULL,
	time TIME NOT NULL
)`)

	testWeight := 220.2
	date := time.Now()

	// Insert a weight for date.
	db.Exec(`INSERT INTO daily_weights (date, time, weight) VALUES ($1, $2, $3)`, date.Format(dateFormat), date.Format(dateFormatTime), testWeight)

	exists, err := checkWeightExists(db, date)
	fmt.Println(exists)
	fmt.Println(err)

	// Output:
	// true
	// <nil>
}

func ExampleUpdateWeightEntry() {
	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	db.MustExec(`CREATE TABLE IF NOT EXISTS daily_weights (
  id INTEGER PRIMARY KEY,
  date DATE NOT NULL,
	time TIME NOT NULL,
  weight REAL NOT NULL
)`)

	testWeight := 220.2
	date := time.Now()

	// Insert a weight for date.
	db.Exec(`INSERT INTO daily_weights (date, time, weight) VALUES ($1, $2, $3)`, date.Format(dateFormat), date.Format(dateFormatTime), testWeight)

	newWeight := 225.2

	err = updateWeightEntry(db, 1, newWeight)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Verify the weight was updated
	var weight float64
	err = db.Get(&weight, `SELECT weight FROM daily_weights WHERE date = ?`, date.Format(dateFormat))

	fmt.Println(weight)
	fmt.Println(err)

	// Output:
	// 225.2
	// <nil>
}

func ExampleDeleteOneWeightEntry() {
	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	db.MustExec(`CREATE TABLE IF NOT EXISTS daily_weights (
  id INTEGER PRIMARY KEY,
  date DATE NOT NULL,
  weight REAL NOT NULL,
	time TIME NOT NULL
)`)

	testWeight := 220.2
	date := time.Now()

	// Insert a weight for date.
	db.Exec(`INSERT INTO daily_weights (date, time, weight) VALUES ($1, $2, $3)`, date.Format(dateFormat), date.Format(dateFormatTime), testWeight)

	err = deleteOneWeightEntry(db, 1)
	if err != nil {
		fmt.Println(err)
		return
	}

	var weight float64
	db.Get(&weight, `SELECT weight FROM daily_weights WHERE date = ?`, date.Format(dateFormat))

	fmt.Println(weight)
	fmt.Println(err)

	// Output:
	// 0
	// <nil>
}

func ExampleUpdateFoodEntry() {
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

	// Create the foods table
	tx.MustExec(` CREATE TABLE IF NOT EXISTS foods (
  food_id INTEGER PRIMARY KEY,
  food_name TEXT NOT NULL,
  serving_size REAL NOT NULL,
  serving_unit TEXT NOT NULL,
  household_serving TEXT NOT NULL
	)`)

	// Insert foods
	tx.MustExec(`INSERT INTO foods (food_id, food_name, serving_size, serving_unit, household_serving) VALUES
	(1, 'Chicken Breast', 100, 'g', '1/2 piece')
	`)

	// Create daily foods table
	tx.MustExec(`CREATE TABLE daily_foods (
  id INTEGER PRIMARY KEY,
  food_id INTEGER REFERENCES foods(food_id) NOT NULL,
  meal_id INTEGER REFERENCES meals(meal_id),
  date DATE NOT NULL,
	time TIME NOT NULL,
  serving_size REAL NOT NULL,
  number_of_servings REAL DEFAULT 1 NOT NULL,
	calories REAL NOT NULL,
  protein REAL NOT NULL,
  fat REAL NOT NULL,
  carbs REAL NOT NULL,
	price REAL DEFAULT 0
)`)

	// Insert daily food entry.
	tx.MustExec(`INSERT INTO daily_foods (food_id, date, time, serving_size, number_of_servings, calories, protein, fat, carbs) VALUES
(1, "2023-01-01", "00:00:00", 100, 1, 56, 5, 4, 5)
	`)

	food := &Food{
		ID:               1,
		Name:             "Chicken Breast",
		ServingUnit:      "g",
		ServingSize:      100,
		HouseholdServing: "1/2 piece",
		NumberOfServings: 2,
		Calories:         112,
		FoodMacros: &FoodMacros{
			Protein: 10,
			Fat:     8,
			Carbs:   10,
		},
	}

	err = updateFoodEntry(tx, 1, *food)
	if err != nil {
		log.Println(err)
		return
	}

	tx.Commit()

	// Verify the food entry was updated
	var numServings float64
	err = db.Get(&numServings, `SELECT number_of_servings FROM daily_foods WHERE date = $1`, "2023-01-01")

	fmt.Println(numServings)
	fmt.Println(err)

	// Output:
	// 2
	// <nil>
}

func ExampleGetRecentFoodEntries() {
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

	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS foods (
		food_id INTEGER PRIMARY KEY,
		food_name TEXT NOT NULL,
		serving_size REAL NOT NULL,
		serving_unit TEXT NOT NULL,
		household_serving TEXT NOT NULL,
		brand_name TEXT,
		cost REAL
	);

	CREATE TABLE IF NOT EXISTS daily_foods (
		id INTEGER PRIMARY KEY,
		food_id INTEGER REFERENCES foods(food_id) NOT NULL,
		meal_id INTEGER REFERENCES meals(meal_id),
		date DATE NOT NULL,
		time TIME NOT NULL,
		serving_size REAL NOT NULL,
		number_of_servings REAL DEFAULT 1 NOT NULL,
		calories REAL NOT NULL,
		protein REAL NOT NULL,
		fat REAL NOT NULL,
		carbs REAL NOT NULL
	);
	`)

	if err != nil {
		fmt.Printf("Failed to build tables: %v\n", err)
		return
	}

	_, err = tx.Exec(`
	INSERT INTO foods VALUES (1, 'Apple', 100, 'g', '1 medium', NULL, NULL);
	INSERT INTO foods VALUES (2, 'Bread', 100, 'g', '1 medium', NULL, NULL);
	INSERT INTO foods VALUES (3, 'Tomato', 100, 'g', '1 medium', NULL, NULL);
	INSERT INTO daily_foods VALUES (1, 1, NULL, '2023-01-01', '00:00:00', 42, 1, 50, 5, 5, 5);
	INSERT INTO daily_foods VALUES (2, 1, NULL, '2023-01-01', '00:00:00', 42, 1, 50, 5, 5, 5);
	INSERT INTO daily_foods VALUES (3, 2, NULL, '2023-01-02', '00:00:00', 42, 1, 50, 5, 5, 5);
	`)
	if err != nil {
		fmt.Println("Failed to insert data:", err)
		return
	}

	dailyFoods, err := getRecentFoodEntries(tx, 10)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, food := range dailyFoods {
		fmt.Println(food.FoodName)
	}

	// Output:
	// Bread
	// Apple
}

func ExampleGetMealsWithRecentFirst() {
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
    CREATE TABLE IF NOT EXISTS meals (
        meal_id INTEGER PRIMARY KEY,
        meal_name TEXT NOT NULL
    );

    CREATE TABLE IF NOT EXISTS daily_meals (
      id INTEGER PRIMARY KEY,
      meal_id INTEGER REFERENCES meals(meal_id),
      date DATE NOT NULL,
      time TIME NOT NULL
    );
  `)

	_, err = tx.Exec(`
	INSERT INTO meals VALUES
	(1, 'Pie'),
	(2, 'Shake'),
	(3, 'Pizza')
	`)
	if err != nil {
		log.Printf("Failed to insert data into the meals table: %v\n", err)
	}

	_, err = tx.Exec(`
	INSERT INTO daily_meals VALUES (1, 3, '2023-01-01', '00:00:00:');
	INSERT INTO daily_meals VALUES (2, 3, '2023-01-01', '00:00:00:');
	INSERT INTO daily_meals VALUES (3, 2, '2023-01-01', '00:00:00:');
	`)
	if err != nil {
		log.Printf("Failed to insert data into the daily meals table: %v\n", err)
	}

	meals, err := getMealsWithRecentFirst(tx)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, meal := range meals {
		fmt.Println(meal.Name)
	}

	// Output:
	// Shake
	// Pizza
	// Pie
}

func ExampleGetMealFoodWithPref() {
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

	// Create tables.
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS foods (
			food_id INTEGER PRIMARY KEY,
			food_name TEXT NOT NULL,
			serving_size REAL NOT NULL,
			serving_unit TEXT NOT NULL,
			household_serving TEXT NOT NULL
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

		CREATE TABLE IF NOT EXISTS nutrients (
			nutrient_id INTEGER PRIMARY KEY,
			nutrient_name TEXT NOT NULL,
			unit_name TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS daily_foods (
  		id INTEGER PRIMARY KEY,
  		food_id INTEGER REFERENCES foods(food_id) NOT NULL,
  		meal_id INTEGER REFERENCES meals(meal_id),
  		date DATE NOT NULL,
			time TIME NOT NULL,
  		serving_size REAL NOT NULL,
  		number_of_servings REAL DEFAULT 1 NOT NULL,
			calories REAL NOT NULL,
			protein REAL NOT NULL,
  		fat REAL NOT NULL,
  		carbs REAL NOT NULL
		);
  `)
	if err != nil {
		log.Fatalf("failed to create schema: %s", err)
	}

	// Insert test data.
	_, err = tx.Exec(`INSERT INTO nutrients (nutrient_id, nutrient_name, unit_name) VALUES
  (1, 'Protein', 'g'),
  (2, 'Total lipid (fat)', 'g'),
  (3, 'Carbohydrate, by difference', 'g'),
  (4, 'Energy', 'KCAL')`)
	if err != nil {
		log.Fatalf("failed to insert data into nutrients: %s", err)
	}

	_, err = tx.Exec(`
  INSERT INTO foods VALUES (1, 'Apple', 150, 'g', '1 medium');
  INSERT INTO food_prefs VALUES (1, 160, 1);
  INSERT INTO meal_food_prefs VALUES (1, 1, 180, 2);
`)
	if err != nil {
		log.Fatalf("failed to insert data into foods, food_prefs, meal_food_prefs: %s", err)
	}

	_, err = tx.Exec(`
  INSERT INTO food_nutrients VALUES (1, 1, 1, 0.3, 71);  -- 0.3g Protein
  INSERT INTO food_nutrients VALUES (2, 1, 2, 0.2, 71);  -- 0.2g Fat
  INSERT INTO food_nutrients VALUES (3, 1, 3, 12, 71);   -- 12g Carbohydrates
  INSERT INTO food_nutrients VALUES (4, 1, 4, 52, 71);   -- 52KCAL Energy
`)
	if err != nil {
		log.Fatalf("failed to insert data into food_nutrients: %s", err)
	}

	// Test getMealFoodWithPref.
	mealFood, err := getMealFoodWithPref(tx, 1, 1)
	if err != nil {
		log.Fatalf("getMealFoodWithPref failed: %s", err)
	}

	err = tx.Commit()
	if err != nil {
		fmt.Printf("Failed to commit transaction: %v\n.", err)
		return
	}

	// Print the result for verification.
	fmt.Printf("Food Name: %s\n", mealFood.Food.Name)
	fmt.Printf("Serving Size: %.2f\n", mealFood.ServingSize)
	fmt.Printf("Number of Servings: %.2f\n", mealFood.NumberOfServings)
	fmt.Printf("Calories: %.2f\n", mealFood.Food.Calories)
	fmt.Println("Macros:")
	fmt.Printf("  - Protein: %.2f\n", mealFood.Food.FoodMacros.Protein)
	fmt.Printf("  - Fat: %.2f\n", mealFood.Food.FoodMacros.Fat)
	fmt.Printf("  - Carbs: %.2f\n", mealFood.Food.FoodMacros.Carbs)

	// Output:
	// Food Name: Apple
	// Serving Size: 180.00
	// Number of Servings: 2.00
	// Calories: 187.20
	// Macros:
	//   - Protein: 1.08
	//   - Fat: 0.72
	//   - Carbs: 43.20
}

func ExampleGetFoodWithPref() {
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

	// Create tables.
	_, err = tx.Exec(`
    CREATE TABLE IF NOT EXISTS foods (
      food_id INTEGER PRIMARY KEY,
      food_name TEXT NOT NULL,
      serving_size REAL NOT NULL,
      serving_unit TEXT NOT NULL,
      household_serving TEXT NOT NULL
    );

    CREATE TABLE IF NOT EXISTS food_prefs (
      food_id INTEGER PRIMARY KEY,
      serving_size REAL,
      number_of_servings REAL DEFAULT 1 NOT NULL,
      FOREIGN KEY(food_id) REFERENCES foods(food_id)
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

    CREATE TABLE IF NOT EXISTS nutrients (
      nutrient_id INTEGER PRIMARY KEY,
      nutrient_name TEXT NOT NULL,
      unit_name TEXT NOT NULL
    );

  `)
	if err != nil {
		log.Fatalf("failed to create schema: %s", err)
	}

	// Insert test data.
	_, err = tx.Exec(`INSERT INTO nutrients (nutrient_id, nutrient_name, unit_name) VALUES
  (1, 'Protein', 'g'),
  (2, 'Total lipid (fat)', 'g'),
  (3, 'Carbohydrate, by difference', 'g'),
  (4, 'Energy', 'KCAL')`)
	if err != nil {
		log.Fatalf("failed to insert data into nutrients: %s", err)
	}

	_, err = tx.Exec(`
  INSERT INTO foods VALUES (1, 'Apple', 100, 'g', '1 medium');
  INSERT INTO food_prefs VALUES (1, 160, 1);
`)
	if err != nil {
		log.Fatalf("failed to insert data into foods, food_prefs: %s", err)
	}

	_, err = tx.Exec(`
  INSERT INTO food_nutrients VALUES (1, 1, 1, 0.3, 71);  -- 0.3g Protein
  INSERT INTO food_nutrients VALUES (2, 1, 2, 0.2, 71);  -- 0.2g Fat
  INSERT INTO food_nutrients VALUES (3, 1, 3, 12, 71);   -- 12g Carbohydrates
  INSERT INTO food_nutrients VALUES (4, 1, 4, 52, 71);   -- 52KCAL Energy
`)
	if err != nil {
		log.Fatalf("failed to insert data into food_nutrients: %s", err)
	}

	// Test getFoodWithPref.
	food, err := getFoodWithPref(tx, 1)
	if err != nil {
		log.Fatalf("getFoodWithPref failed: %s", err)
	}

	err = tx.Commit()
	if err != nil {
		fmt.Printf("Failed to commit transaction: %v\n.", err)
		return
	}

	// Print the result for verification.
	fmt.Printf("Food Name: %s\n", food.Name)
	fmt.Printf("Serving Size: %.2f\n", food.ServingSize)
	fmt.Printf("Number of Servings: %.2f\n", food.NumberOfServings)
	fmt.Printf("Calories: %.2f\n", food.Calories)
	fmt.Println("Macros:")
	fmt.Printf("  - Protein: %.2f\n", food.FoodMacros.Protein)
	fmt.Printf("  - Fat: %.2f\n", food.FoodMacros.Fat)
	fmt.Printf("  - Carbs: %.2f\n", food.FoodMacros.Carbs)

	// Output:
	// Food Name: Apple
	// Serving Size: 160.00
	// Number of Servings: 1.00
	// Calories: 83.20
	// Macros:
	//   - Protein: 0.48
	//   - Fat: 0.32
	//   - Carbs: 19.20
}

func ExampleAddMealEntry() {
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
		CREATE TABLE IF NOT EXISTS meals (
				meal_id INTEGER PRIMARY KEY,
				meal_name TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS daily_meals (
  		id INTEGER PRIMARY KEY,
  		meal_id INTEGER REFERENCES meals(meal_id),
  		date DATE NOT NULL,
			time TIME NOT NULL
		);
	`)

	_, err = tx.Exec(`INSERT INTO meals VALUES (1, 'Pie')`)
	if err != nil {
		log.Printf("Failed to insert data into meal table: %v\n", err)
	}

	date := time.Now()
	meal := Meal{
		ID:   1,
		Name: "Pie",
	}

	err = addMealEntry(tx, meal, date)
	if err != nil {
		log.Printf("Failed to add meal entry: %v\n", err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Printf("Failed to commit transaction: %v\n", err)
		return
	}

	// Verify the meal was logged correctly.
	var mealID float64
	err = db.Get(&mealID, `SELECT meal_id FROM daily_meals WHERE date = ?`, date.Format(dateFormat))
	if err != nil {
		log.Printf("Failed to get meal id from daily meals: %v\n", err)
		return
	}

	fmt.Println(mealID)
	fmt.Println(err)

	// Output:
	// 1
	// <nil>
}

func ExampleAddMealFoodEntries() {
	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		log.Println("Could not connect to test database:", err)
	}
	defer db.Close()

	// Start a new transaction
	tx, err := db.Beginx()
	if err != nil {
		return
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // re-throw panic
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// Create the foods table
	tx.MustExec(`
	CREATE TABLE IF NOT EXISTS foods (
  food_id INTEGER PRIMARY KEY,
  food_name TEXT NOT NULL,
  serving_size REAL NOT NULL,
  serving_unit TEXT NOT NULL,
  household_serving TEXT NOT NULL
  );

	CREATE TABLE daily_foods (
  id INTEGER PRIMARY KEY,
  food_id INTEGER REFERENCES foods(food_id) NOT NULL,
  meal_id INTEGER REFERENCES meals(meal_id),
  date DATE NOT NULL,
	time TIME NOT NULL,
  serving_size REAL NOT NULL,
  number_of_servings REAL DEFAULT 1 NOT NULL,
	calories REAL NOT NULL,
  protein REAL NOT NULL,
  fat REAL NOT NULL,
  carbs REAL NOT NULL,
	price REAL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS meals (
  meal_id INTEGER PRIMARY KEY,
  meal_name TEXT NOT NULL
  );

  CREATE TABLE IF NOT EXISTS daily_meals (
  id INTEGER PRIMARY KEY,
  meal_id INTEGER REFERENCES meals(meal_id),
  date DATE NOT NULL,
	time TIME NOT NULL
  );
	`)
	if err != nil {
		log.Printf("failed to create schema: %v\n", err)
		return
	}

	// Insert foods
	tx.MustExec(`INSERT INTO foods (food_id, food_name, serving_size, serving_unit, household_serving) VALUES
  (1, 'Chicken Breast', 100, 'g', '1/2 piece'),
  (2, 'Rice', 100, 'g', '1/2 cup'),
	(3, 'Broccoli', 156, 'g', '1 cup')
  `)

	// Insert meal
	tx.MustExec(`INSERT INTO meals (meal_id, meal_name) VALUES
  (1, 'Chicken, rice, and broccoli')
  `)

	mealFoods := []*MealFood{
		{
			Food: Food{
				ID:               1,
				Name:             "Chicken Breast",
				ServingSize:      100,
				ServingUnit:      "g",
				HouseholdServing: "1/2 piece",
				Calories:         165,
				FoodMacros: &FoodMacros{
					Protein: 31,
					Fat:     3.6,
					Carbs:   0,
				},
			},
			NumberOfServings: 1,
			ServingSize:      100,
		},
		{
			Food: Food{
				ID:               2,
				Name:             "Rice",
				ServingSize:      100,
				ServingUnit:      "g",
				HouseholdServing: "1/2 cup",
				Calories:         130,
				FoodMacros: &FoodMacros{
					Protein: 2.6,
					Fat:     0.3,
					Carbs:   28,
				},
			},
			NumberOfServings: 1,
			ServingSize:      100,
		},
		{
			Food: Food{
				ID:               3,
				Name:             "Broccoli",
				ServingSize:      156,
				ServingUnit:      "g",
				HouseholdServing: "1 cup",
				Calories:         50,
				FoodMacros: &FoodMacros{
					Protein: 4,
					Fat:     0.6,
					Carbs:   10,
				},
			},
			NumberOfServings: 1,
			ServingSize:      156,
		},
	}

	testDate := time.Date(2023, 7, 15, 0, 0, 0, 0, time.UTC)
	err = addMealFoodEntries(tx, 1, mealFoods, testDate)
	if err != nil {
		log.Printf("Failed to add meal food entries: %v\n.", err)
		return
	}

	var foodIDs []int
	err = tx.Select(&foodIDs, `SELECT food_id FROM daily_foods WHERE meal_id = 1`)
	if err != nil {
		log.Printf("failed to fetch food_ids: %v\n", err)
		return
	}

	/*
		err = tx.Commit()
		if err != nil {
			log.Printf("Failed to commit transaction: %v\n.", err)
			return
		}
	*/

	for _, id := range foodIDs {
		fmt.Println(id)
	}
	fmt.Println(err)

	// Output:
	// 1
	// 2
	// 3
	// <nil>
}

func ExampleGetValidLog() {
	entries := &[]Entry{
		{
			UserWeight: 70.5,
			Calories:   2000,
			Date:       time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
			Protein:    150, // Assuming 30% of calories from protein for this entry
			Carbs:      200, // Assuming 40% of calories from carbs for this entry
			Fat:        67,  // Assuming 30% of calories from fat for this entry
		},
		{
			UserWeight: 70.5,
			Calories:   2000,
			Date:       time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC),
			Protein:    150, // Assuming 30% of calories from protein for this entry
			Carbs:      200, // Assuming 40% of calories from carbs for this entry
			Fat:        67,  // Assuming 30% of calories from fat for this entry
		},
		{
			UserWeight: 70.1,
			Calories:   1900,
			Date:       time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC),
			Protein:    142.5, // Assuming 30% of calories from protein for this entry
			Carbs:      190,   // Assuming 40% of calories from carbs for this entry
			Fat:        63.3,  // Assuming 30% of calories from fat for this entry
		},
		{
			UserWeight: 69.8,
			Calories:   1850,
			Date:       time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC),
			Protein:    138.75, // Assuming 30% of calories from protein for this entry
			Carbs:      185,    // Assuming 40% of calories from carbs for this entry
			Fat:        61.6,   // Assuming 30% of calories from fat for this entry
		},
	}

	today := time.Now()
	u := UserInfo{}
	u.Phase.StartDate = (*entries)[1].Date
	u.Phase.EndDate = today.AddDate(0, 0, 7)

	subset := GetValidLog(&u, entries)
	fmt.Println("Weight Calories Date")
	for _, entry := range *subset {
		fmt.Println(entry.UserWeight, entry.Calories, entry.Date)
	}

	// Output:
	// Weight Calories Date
	// 70.5 2000 2023-01-03 00:00:00 +0000 UTC
	// 70.1 1900 2023-01-05 00:00:00 +0000 UTC
	// 69.8 1850 2023-01-07 00:00:00 +0000 UTC
}

func ExampleUpdateFoodPrefs() {
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
	defer tx.Rollback()

	// Create the food_nutrients table
	tx.MustExec(`
      CREATE TABLE IF NOT EXISTS foods (
        food_id INTEGER PRIMARY KEY,
        food_name TEXT NOT NULL,
        serving_size REAL NOT NULL,
        serving_unit TEXT NOT NULL,
        household_serving TEXT NOT NULL
      );

			CREATE TABLE IF NOT EXISTS food_prefs (
				food_id INTEGER PRIMARY KEY,
				serving_size REAL,
				number_of_servings REAL DEFAULT 1 NOT NULL,
				FOREIGN KEY(food_id) REFERENCES foods(food_id)
			);
	`)

	// Insert food
	tx.MustExec(`INSERT INTO foods (food_id, food_name, serving_size, serving_unit, household_serving) VALUES
  (1, 'Chicken Breast', 100, 'g', '1/2 piece')
  `)

	pref := &FoodPref{}
	pref.FoodID = 1
	pref.ServingSize = 300
	pref.NumberOfServings = 1.2

	err = updateFoodPrefs(tx, pref)
	if err != nil {
		log.Printf("Failed to update food prefs: %v\n", err)
		return
	}

	var p FoodPref
	err = tx.Get(&p, `SELECT serving_size, number_of_servings FROM food_prefs WHERE food_id = 1`)
	if err != nil {
		log.Println(err)
		return
	}

	tx.Commit()

	fmt.Println("Updated serving size:", p.ServingSize)
	fmt.Println("Updated number of servings:", p.NumberOfServings)
	fmt.Println(err)

	// Output:
	// Updated serving size: 300
	// Updated number of servings: 1.2
	// <nil>
}

func ExampleGetTotalFoodsLogged() {
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

	// Create the daily_foods table
	tx.MustExec(`CREATE TABLE daily_foods (
  id INTEGER PRIMARY KEY,
  food_id INTEGER REFERENCES foods(food_id) NOT NULL,
  meal_ID INTEGER REFERENCES meals(meal_id),
  date DATE NOT NULL,
  time TIME NOT NULL,
  number_of_servings REAL DEFAULT 1 NOT NULL
  )`)

	// Insert data
	tx.MustExec(`INSERT INTO daily_foods (food_id, date, time, number_of_servings) VALUES
	(1, '2023-01-01', '00:00:00', 1),
	(2, '2023-01-01', '00:00:00', 1),
	(2, '2023-01-02','00:00:00',  1),
	(4, '2023-01-02','00:00:00',  1),
	(5, '2023-01-03', '00:00:00', 1),
	(1, '2023-01-03', '00:00:00', 1),
	(3, '2023-01-04', '00:00:00', 1),
	(4, '2023-01-04', '00:00:00', 1)
	`)

	total, err := getTotalFoodsLogged(tx)
	if err != nil {
		log.Printf("ExampleGetTotalFoodsLogged failed to get total foods logged: %v\n", err)
		return
	}

	fmt.Println(total)

	tx.Commit()

	// Output:
	// 8
}

func ExampleGetFrequentFoods() {
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

	// Create the foods table
	tx.MustExec(`
  CREATE TABLE IF NOT EXISTS foods (
  food_id INTEGER PRIMARY KEY,
  food_name TEXT NOT NULL,
  serving_size REAL NOT NULL,
  serving_unit TEXT NOT NULL,
  household_serving TEXT NOT NULL
  );

  CREATE TABLE daily_foods (
  id INTEGER PRIMARY KEY,
  food_id INTEGER REFERENCES foods(food_id) NOT NULL,
  meal_id INTEGER REFERENCES meals(meal_id),
  date DATE NOT NULL,
  time TIME NOT NULL,
  serving_size REAL NOT NULL,
  number_of_servings REAL DEFAULT 1 NOT NULL
  );
	`)

	// Insert foods
	tx.MustExec(`
		INSERT INTO foods (food_id, food_name, serving_size, serving_unit, household_serving) VALUES
  	(1, 'Chicken', 100, 'g', '1/2 piece'),
  	(2, 'Beef', 100, 'g', '1/2 cup'),
  	(3, 'Pork', 100, 'g', '1 cup')
  `)

	// Insert daily foods
	tx.MustExec(`
		INSERT INTO daily_foods (food_id, date, time ,serving_size) VALUES
		(1, '2023-07-10', '00:00:00', 100),
		(1, '2023-07-10','00:00:00',  100),
		(1, '2023-07-10', '00:00:00', 100),
		(2, '2023-07-10', '00:00:00', 200),
		(2, '2023-07-11', '00:00:00', 200),
		(3, '2023-07-12', '00:00:00', 300);
	`)

	// Call the function to test
	foods, err := getFrequentFoods(tx, 2)
	if err != nil {
		log.Printf("Failed to get frequent foods: %v\n", err)
		return
	}

	// Print the results
	for _, food := range foods {
		fmt.Printf("%s: %d times\n", food.FoodName, food.Count)
	}

	// Output:
	// Chicken: 3 times
	// Beef: 2 times
}

func ExampleGetFoodEntriesForDate() {
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

	// Create tables.
	_, err = tx.Exec(`
    CREATE TABLE IF NOT EXISTS foods (
      food_id INTEGER PRIMARY KEY,
      food_name TEXT NOT NULL,
      serving_size REAL NOT NULL,
      serving_unit TEXT NOT NULL,
      household_serving TEXT NOT NULL
    );
		CREATE TABLE IF NOT EXISTS daily_foods (
      id INTEGER PRIMARY KEY,
      food_id INTEGER REFERENCES foods(food_id) NOT NULL,
      meal_id INTEGER REFERENCES meals(meal_id),
      date DATE NOT NULL,
      time TIME NOT NULL,
      serving_size REAL NOT NULL,
      number_of_servings REAL DEFAULT 1 NOT NULL,
      calories REAL NOT NULL,
      protein REAL NOT NULL,
      fat REAL NOT NULL,
      carbs REAL NOT NULL,
			price REAL DEFAULT 0
    );
  `)

	if err != nil {
		fmt.Println("Failed to setup tables:", err)
		return
	}

	// Insert foods
	tx.MustExec(`
    INSERT INTO foods (food_id, food_name, serving_size, serving_unit, household_serving) VALUES
    (1, 'Chicken', 100, 'g', '1/2 piece'),
    (2, 'Beef', 100, 'g', '1/2 cup'),
    (3, 'Pork', 100, 'g', '1 cup')
  `)

	// Note: 5th day user did not log any foods.
	tx.MustExec(`INSERT INTO daily_foods (food_id, date, time, serving_size, number_of_servings, calories, protein, fat, carbs) VALUES
    (1, '2023-01-01', '00:00:00', 100, 1, 165, 31, 3.6, 0),
    (1, '2023-01-01', '00:00:00', 100, 1, 165, 31, 3.6, 0),
    (2, '2023-01-01', '00:00:00', 100, 1, 216, 12, 12, 15),
    (1, '2023-01-01', '00:00:00', 100, 1, 165, 31, 3.6, 0),
    (2, '2023-01-03', '00:00:00', 100, 1, 216, 12, 12, 15),
    (2, '2023-01-03', '00:00:00', 100, 1, 165, 31, 3.6, 0),
    (3, '2023-01-04', '00:00:00', 100, 1, 122, 2.73, 0.96, 25.5)
  `)

	date := time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)
	entries, err := getFoodEntriesForDate(tx, date)
	if err != nil {
		log.Println(err)
		return
	}

	for i, entry := range entries {
		fmt.Printf("Entry %d: %s\n", i, entry.FoodName)
	}

	// Output:
	// Entry 0: Chicken
	// Entry 1: Chicken
	// Entry 2: Beef
	// Entry 3: Chicken
}

func ExampleRenderProgressBar() {
	fmt.Println(renderProgressBar(10, 100))

	// Output:
	// [█▒▒▒▒▒▒▒▒▒]
}

func ExampleRenderProgressBar_full() {
	fmt.Println(renderProgressBar(110, 100))

	// Output:
	// [███████████]
}
