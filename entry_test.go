package calories

import (
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rocketlaunchr/dataframe-go"
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
  number_of_servings REAL DEFAULT 1 NOT NULL
	)`)

	// Note: 5th day user did not log any foods.
	db.MustExec(`INSERT INTO daily_foods (food_id, date, number_of_servings) VALUES
	(1, '2023-01-01', 1),
	(2, '2023-01-01', 1),
	(2, '2023-01-02', 1),
	(4, '2023-01-02', 1),
	(5, '2023-01-03', 1),
	(1, '2023-01-03', 1),
	(3, '2023-01-04', 1),
	(4, '2023-01-04', 1)
	`)

	// Create the daily_weights table
	db.MustExec(`CREATE TABLE daily_weights (
  id INTEGER PRIMARY KEY,
  date DATE NOT NULL,
  weight REAL NOT NULL
	)`)

	db.MustExec(`INSERT INTO daily_weights (date, weight) VALUES
	('2023-01-01', 180),
	('2023-01-02', 181),
	('2023-01-03', 182),
	('2023-01-04', 183),
	('2023-01-05', 184)
	`)

	db.MustExec(`CREATE TABLE IF NOT EXISTS daily_meals (
  id INTEGER PRIMARY KEY,
  meal_id INTEGER REFERENCES meals(meal_id),
  date DATE NOT NULL
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

	// Get all entries
	entries, err := GetAllEntries(db)
	if err != nil {
		panic(err)
	}

	for _, entry := range *entries {
		fmt.Println("Date: ", entry.Date.Format(dateFormat))
		fmt.Println("Weight: ", entry.UserWeight)
		fmt.Println("Calories: ", entry.UserCals)
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
  weight REAL NOT NULL
)`)

	testWeight := 220.2
	date := time.Now()

	// Insert a weight for date.
	db.Exec(`INSERT INTO daily_weights (date, weight) VALUES (?, ?)`, date.Format(dateFormat), testWeight)

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
  weight REAL NOT NULL
)`)

	testWeight := 220.2
	date := time.Now()

	// Insert a weight for date.
	db.Exec(`INSERT INTO daily_weights (date, weight) VALUES (?, ?)`, date.Format(dateFormat), testWeight)

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
  weight REAL NOT NULL
)`)

	testWeight := 220.2
	date := time.Now()

	// Insert a weight for date.
	db.Exec(`INSERT INTO daily_weights (date, weight) VALUES (?, ?)`, date.Format(dateFormat), testWeight)

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
  weight REAL NOT NULL
)`)

	testWeight := 220.2
	date := time.Now()

	// Insert a weight for date.
	db.Exec(`INSERT INTO daily_weights (date, weight) VALUES (?, ?)`, date.Format(dateFormat), testWeight)

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
  serving_size REAL NOT NULL,
  number_of_servings REAL DEFAULT 1 NOT NULL
)`)

	// Insert daily food entry.
	tx.MustExec(`INSERT INTO daily_foods (food_id, date, serving_size) VALUES
	(1, "2023-01-01", 100)
	`)

	pref := &FoodPref{
		FoodID:           1,
		ServingSize:      100,
		NumberOfServings: 2,
	}

	err = updateFoodEntry(tx, 1, *pref)
	if err != nil {
		log.Println(err)
		return
	}

	tx.Commit()

	// Verify the food entry was updated
	var numServings float64
	err = db.Get(&numServings, `SELECT number_of_servings FROM daily_foods WHERE date = ?`, "2023-01-01")

	fmt.Println(numServings)
	fmt.Println(err)

	// Output:
	// 2
	// <nil>
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
  		serving_size REAL NOT NULL,
  		number_of_servings REAL DEFAULT 1 NOT NULL
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

	tx.Commit()

	// Print the result for verification.
	fmt.Printf("Food Name: %s\n", mealFood.Food.Name)
	fmt.Printf("Serving Size: %.2f\n", mealFood.ServingSize)
	fmt.Printf("Number of Servings: %.2f\n", mealFood.NumberOfServings)
	fmt.Printf("Calories: %.2f\n", mealFood.Food.PortionCals)
	fmt.Println("Macros:")
	fmt.Printf("  - Protein: %.2f\n", mealFood.Food.FoodMacros.Protein)
	fmt.Printf("  - Fat: %.2f\n", mealFood.Food.FoodMacros.Fat)
	fmt.Printf("  - Carbs: %.2f\n", mealFood.Food.FoodMacros.Carbs)

	// Output:
	// Food Name: Apple
	// Serving Size: 180.00
	// Number of Servings: 2.00
	// Calories: 124.80
	// Macros:
	//   - Protein: 0.72
	//   - Fat: 0.48
	//   - Carbs: 28.80
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
  		date DATE NOT NULL
		);
	`)

	_, err = tx.Exec(`INSERT INTO meals VALUES (1, 'Pie')`)
	if err != nil {
		log.Fatalf("failed to insert data into meal table: %s", err)
	}

	date := time.Now()
	meal := Meal{
		ID:   1,
		Name: "Pie",
	}

	err = addMealEntry(tx, meal, date)
	if err != nil {
		log.Println(err)
		return
	}

	tx.Commit()

	// Verify the meal was logged correctly.
	var mealID float64
	err = db.Get(&mealID, `SELECT meal_id FROM daily_meals WHERE date = ?`, date.Format(dateFormat))

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
		panic(err)
	}
	defer db.Close()

	// Create the foods table
	db.MustExec(`
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
  serving_size REAL NOT NULL,
  number_of_servings REAL DEFAULT 1 NOT NULL
	);

	CREATE TABLE IF NOT EXISTS meals (
  meal_id INTEGER PRIMARY KEY,
  meal_name TEXT NOT NULL
  );

  CREATE TABLE IF NOT EXISTS daily_meals (
  id INTEGER PRIMARY KEY,
  meal_id INTEGER REFERENCES meals(meal_id),
  date DATE NOT NULL
  );
	`)
	if err != nil {
		log.Fatalf("failed to create schema: %s", err)
	}

	// Insert foods
	db.MustExec(`INSERT INTO foods (food_id, food_name, serving_size, serving_unit, household_serving) VALUES
  (1, 'Chicken Breast', 100, 'g', '1/2 piece'),
  (2, 'Rice', 100, 'g', '1/2 cup'),
	(3, 'Broccoli', 156, 'g', '1 cup')
  `)

	// Insert meal
	db.MustExec(`INSERT INTO meals (meal_id, meal_name) VALUES
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
			},
			NumberOfServings: 1,
			ServingSize:      156,
		},
	}

	testDate := time.Date(2023, 7, 15, 0, 0, 0, 0, time.UTC)
	tx, _ := db.Beginx()
	err = addMealFoodEntries(tx, 1, mealFoods, testDate)

	var foodIDs []int
	err = db.Select(&foodIDs, `SELECT food_id FROM daily_foods WHERE meal_id = 1`)
	if err != nil {
		log.Fatalf("failed to fetch food_ids: %s", err)
	}

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

func ExampleSubset() {
	s1 := dataframe.NewSeriesString("weight", nil, "170", "170", "170", "170", "170", "170", "170", "170")
	s2 := dataframe.NewSeriesString("calories", nil, "2400", "2400", "2400", "2400", "2400", "2400", "2400", "2400")
	s3 := dataframe.NewSeriesString("date", nil, "2023-01-01", "2023-01-02", "2023-01-03", "2023-01-04", "2023-01-05", "2023-01-06", "2023-01-07", "2023-01-08")
	df := dataframe.NewDataFrame(s1, s2, s3)

	indices := []int{0, 2, 4} // indices we're interested in

	s := Subset(df, indices)
	fmt.Println(s)

	// Output:
	// +-----+--------+----------+------------+
	// |     | WEIGHT | CALORIES |    DATE    |
	// +-----+--------+----------+------------+
	// | 0:  |  170   |   2400   | 2023-01-01 |
	// | 1:  |  170   |   2400   | 2023-01-03 |
	// | 2:  |  170   |   2400   | 2023-01-05 |
	// +-----+--------+----------+------------+
	// | 3X3 | STRING |  STRING  |   STRING   |
	// +-----+--------+----------+------------+
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
	// If anything goes wrong, rollback the transaction
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
		log.Println(err)
		return
	}

	tx.Commit()

	var p FoodPref
	err = tx.Get(&p, `SELECT serving_size, number_of_servings FROM food_prefs WHERE food_id = 1`)
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println("Updated serving size:", p.ServingSize)
	fmt.Println("Updated number of servings:", p.NumberOfServings)
	fmt.Println(err)

	// Output:
	// Updated serving size: 300
	// Updated number of servings: 1.2
	// <nil>
}

/*
func ExampleGetValidLogIndices() {
  u := UserInfo{}

  var weightSeriesElements []interface{}
  var caloriesSeriesElements []interface{}
  var dateSeriesElements []interface{}

  weightVal := 184.0
  caloriesVal := 2300.0

  today := time.Now()

	// Initialize dataframe with 18 days prior to today.
  for i := 0; i < 18; i++ {
    dateVal := today.AddDate(0, 0, -17+i).Format(dateFormat)
    dateSeriesElements = append(dateSeriesElements, dateVal)

    weightVal -= 0.10
    weightSeriesElements = append(weightSeriesElements, strconv.FormatFloat(weightVal, 'f', 1, 64))

    caloriesVal -= 10
    caloriesSeriesElements = append(caloriesSeriesElements, strconv.Itoa(int(caloriesVal)))
  }

  weightSeries := dataframe.NewSeriesString("weight", nil, weightSeriesElements...)
  caloriesSeries := dataframe.NewSeriesString("calories", nil, caloriesSeriesElements...)
  dateSeries := dataframe.NewSeriesString("date", nil, dateSeriesElements...)

  logs := dataframe.NewDataFrame(weightSeries, caloriesSeries, dateSeries)

	// Set starting date a few indices past 0; this simulates entries that
	// were logged before a diet phase began.
  u.Phase.StartDate, _ = time.Parse(dateFormat, logs.Series[dateCol].Value(10).(string))
	// Set end date to some arbitrary point past last logged entry (today)
  u.Phase.EndDate = today.AddDate(0, 0, 7)
  u.Phase.Active = true

  fmt.Println(getValidLogIndices(&u, logs))

  // Output:
  // [10 11 12 13 14 15 16]
}
*/
