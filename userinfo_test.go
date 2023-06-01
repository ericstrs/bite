package calories

import "fmt"

func ExampleMifflin() {
	weight := 80.0  // kg
	height := 180.0 // cm
	age := 30
	gender := "male"
	result := Mifflin(weight, height, age, gender)
	fmt.Println(result)

	// Output:
	// 1780
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

	setMinMacros(u)
	fmt.Println("Minimum daily protein:", u.Macros.MinProtein)

	// Output:
	// Minimum daily protein: 52
	//
}

// TODO: make example tests for when weight has not been set, weight
// is negative.
