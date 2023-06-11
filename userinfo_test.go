package calories

import "fmt"

func ExampleMifflin() {
	u := UserInfo{
		Weight: 180.0, // lbs
		Height: 180.0, // cm
		Age:    30,
		Sex:    "male",
	}

	result := Mifflin(&u)
	fmt.Println(result)

	// Output:
	// 1796.466266
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

// TODO: make example tests for when weight has not been set, weight
// is negative.
