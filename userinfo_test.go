package calories

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
)

func ExampleReadConfig() {
	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS config (
			user_id INTEGER PRIMARY KEY,
			sex TEXT NOT NULL,
			weight REAL NOT NULL,
			height REAL NOT NULL,
			age INTEGER NOT NULL,
			activity_level TEXT NOT NULL,
			tdee REAL NOT NULL,
			system TEXT NOT NULL,
			macros_id INTEGER,
			phase_id INTEGER,
			FOREIGN KEY (macros_id) REFERENCES macros(macros_id),
			FOREIGN KEY (phase_id) REFERENCES phase_info(phase_id)
		);

		CREATE TABLE IF NOT EXISTS macros (
				macros_id INTEGER PRIMARY KEY,
				protein REAL NOT NULL,
				min_protein REAL NOT NULL,
				max_protein REAL NOT NULL,
				carbs REAL NOT NULL,
				min_carbs REAL NOT NULL,
				max_carbs REAL NOT NULL,
				fats REAL NOT NULL,
				min_fats REAL NOT NULL,
				max_fats REAL NOT NULL
		);

		CREATE TABLE IF NOT EXISTS phase_info (
				phase_id INTEGER PRIMARY KEY,
				user_id INTEGER NOT NULL,
				name TEXT NOT NULL,
				goal_calories REAL NOT NULL,
				start_weight REAL NOT NULL,
				goal_weight REAL NOT NULL,
				weight_change_threshold REAL NOT NULL,
				weekly_change REAL NOT NULL,
				start_date DATE NOT NULL,
				end_date DATE NOT NULL,
				last_checked_week DATE NOT NULL,
				duration REAL NOT NULL,
				max_duration REAL NOT NULL,
				min_duration REAL NOT NULL,
				status TEXT NOT NULL CHECK(status IN ('active', 'completed', 'paused', 'stopped', 'scheduled')),
				FOREIGN KEY (user_id) REFERENCES config(user_id)
		);
	`)

	if err != nil {
		log.Println("Failed to setup tables:", err)
		return
	}

	// Insert dummy data
	_, err = db.Exec(`
    INSERT INTO macros (protein, min_protein, max_protein, carbs, min_carbs, max_carbs, fats, min_fats, max_fats)
    VALUES (100, 90, 110, 200, 180, 220, 50, 45, 55);

    INSERT INTO phase_info (user_id, name, goal_calories, start_weight, goal_weight, weight_change_threshold, weekly_change, start_date, end_date, last_checked_week, duration, max_duration, min_duration, status)
    VALUES (1, 'Weight Loss', 2000, 190, 170, 2, -1, '2023-01-01', '2023-04-01', '2023-01-07', 12, 16, 8, 'active');

    INSERT INTO config (sex, weight, height, age, activity_level, tdee, system, macros_id, phase_id)
    VALUES ('M', 190, 175, 30, 'Moderate', 2500, 'Imperial', 1, 1);
    `)
	if err != nil {
		log.Printf("Failed to insert dummy data: %v", err)
		return
	}

	// Call the ReadConfig function
	u, err := ReadConfig(db)
	if err != nil {
		log.Printf("Failed to read config: %v", err)
		return
	}

	fmt.Println("UserID:", u.UserID)
	fmt.Println("Protein:", u.Macros.Protein)
	fmt.Println("PhaseID:", u.Phase.PhaseID)

	// Output:
	// UserID: 1
	// Protein: 100
	// PhaseID: 1
}

func ExampleMifflin() {
	u := UserInfo{
		Weight: 180.0,    // lbs
		Height: 70.86614, // inches
		Age:    30,
		Sex:    "male",
	}

	result := Mifflin(&u)
	fmt.Printf("%.1f\n", result)

	// Output:
	// 1796.5
}

func ExampleUnknownActivity() {
	a := "unknown"
	_, err := activity(a)
	fmt.Println(err)

	// Output:
	// unknown activity level: unknown
}

func ExampleTDEE() {
	bmr := 1780.0
	activityLevel := "active"
	tdee := TDEE(bmr, activityLevel)
	fmt.Println(tdee)

	// Output:
	// 3070.5
}

func ExampleLbsToKg() {
	r := lbsToKg(10)
	fmt.Println(r)

	// Output:
	// 4.535923700000001
}

func ExampleCalculateMacros() {
	u := UserInfo{
		Weight: 180,
		TDEE:   2700,
	}
	u.Phase.GoalCalories = 2400

	setMinMaxMacros(&u)

	protein, carbs, fat := calculateMacros(&u)
	fmt.Println("Protein:", protein)
	fmt.Println("Carbs:", carbs)
	fmt.Println("Fat:", fat)

	// Output:
	// Protein: 180
	// Carbs: 270
	// Fat: 66.67
}

func ExampleCalculateMacros_extremeCut() {
	u := UserInfo{
		Weight: 180,
		TDEE:   2700,
	}
	u.Phase.GoalCalories = 1200

	setMinMaxMacros(&u)

	protein, carbs, fat := calculateMacros(&u)
	fmt.Println("Protein:", protein)
	fmt.Println("Carbs:", carbs)
	fmt.Println("Fat:", fat)

	// Output:
	// Fats are below minimum limit. Taking calories from carbs and moving them to fats.
	// Minimum carb limit reached and fats are still under minimum amount. Attempting to take calories from protein and move them to fats.
	// Protein: 124.49999999999999
	// Carbs: 54
	// Fat: 54
}

func ExampleCalculateMacros_exceedCutCals() {
	u := UserInfo{
		Weight: 180,
		TDEE:   2700,
	}
	u.Phase.GoalCalories = 900

	setMinMaxMacros(&u)

	protein, carbs, fat := calculateMacros(&u)
	fmt.Println("Protein:", protein)
	fmt.Println("Carbs:", carbs)
	fmt.Println("Fat:", fat)

	// Output:
	// Minimum macro values in calories exceed original calorie goal of 900.00
	// New daily calorie goal: 918
	// Protein: 54
	// Carbs: 54
	// Fat: 54
}

func ExampleCalculateMacros_extremeBulk() {
	u := UserInfo{
		Weight: 180,
		TDEE:   2700,
	}
	u.Phase.GoalCalories = 6000

	setMinMaxMacros(&u)

	protein, carbs, fat := calculateMacros(&u)
	fmt.Println("Protein:", protein)
	fmt.Println("Carbs:", carbs)
	fmt.Println("Fat:", fat)

	// Output:
	// Calculated fats are above maximum amount. Taking calories from fats and moving them to carbs.
	// Carb maximum limit reached and fats are still over maximum amount. Attempting to take calories from fats and move them to protein.
	// Protein: 180
	// Carbs: 720
	// Fat: 266.6666666666667
}

func ExampleSetMinMaxMacros() {
	u := UserInfo{
		Weight: 180, // lbs
	}
	u.Phase.GoalCalories = 2000

	setMinMaxMacros(&u)
	fmt.Println("Minimum daily protein:", u.Macros.MinProtein)
	fmt.Println("Maximum daily protein:", u.Macros.MaxProtein)
	fmt.Println("Minimum daily carbs:", u.Macros.MinCarbs)
	fmt.Println("Maximum daily carbs:", u.Macros.MaxCarbs)
	fmt.Println("Minimum daily fat:", u.Macros.MinFats)
	fmt.Println("Maximum daily fat:", u.Macros.MaxFats)

	// Output:
	// Minimum daily protein: 54
	// Maximum daily protein: 360
	// Minimum daily carbs: 54
	// Maximum daily carbs: 720
	// Minimum daily fat: 54
	// Maximum daily fat: 88.88888888888889
}

func ExampleValidateSystem() {
	err := validateSystem("1")
	fmt.Println(err)

	// Output:
	// <nil>
}

func ExampleValidateSex() {
	err := validateSex("male")
	fmt.Println(err)

	// Output:
	// <nil>
}

func ExampleValidateSex_error() {
	err := validateSex("foo")
	fmt.Println(err)

	// Output:
	// Invalid sex.
}

func ExampleValidateAge() {
	a, err := validateAge("30")
	fmt.Println(a)
	fmt.Println(err)

	// Output:
	// 30
	// <nil>
}

func ExampleValidateAge_error() {
	a, err := validateAge("foo")
	fmt.Println(a)
	fmt.Println(err)

	// Output:
	// 0
	// Invalid age.
}

func ExampleValidateActivity() {
	err := validateActivity("very")
	fmt.Println(err)

	// Output:
	// <nil>
}
