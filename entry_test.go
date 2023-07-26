package calories

import (
	"fmt"
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

func ExampleAddWeightLog() {
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

	err = addWeightLog(db, date, testWeight)
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

func ExampleAddWeightLog_exists() {
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
	err = addWeightLog(db, date, testWeight)
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
