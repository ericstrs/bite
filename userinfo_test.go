package calories

import "fmt"

func ExampleMifflin() {
	u := UserInfo{
		Weight: 180.0, // lbs
		Height: 180.0, // cm
		Age:    30,
		Gender: "male",
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
	weight := 180.0 // lbs
	fatPercent := 0.4

	protein, carbs, fat := CalculateMacros(weight, fatPercent)
	fmt.Println("Protein:", protein)
	fmt.Println("Carbs:", carbs)
	fmt.Println("Fat:", fat)

	// Output:
	// Protein: 180
	// Carbs: 342
	// Fat: 72
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
	// Maximum daily carbs: 900
	// Minimum daily fat: 54
	// Maximum daily fat: 88.88888888888889
}

// TODO: make example tests for when weight has not been set, weight
// is negative.
