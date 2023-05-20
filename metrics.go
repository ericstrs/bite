// Calories package provides dietary insight.
package calories

import "fmt"

// activity returns the scale based on the user's activity level.
func activity(a string) (float64, error) {
	activityMap := map[string]float64{
		"sedentary": 1.2,
		"light":     1.375,
		"moderate":  1.55,
		"active":    1.725,
		"very":      1.9,
	}

	value, f := activityMap[a]
	if !f {
		return -1, fmt.Errorf("unknown activity level: %s", a)
	}

	return value, nil
}

// Mifflin calculates and returnsthe Basal Metabolic Rate (BMR) which is based on
// weight (kg), height (cm), age (years), and gender.
func Mifflin(weight, height float64, age int, gender string) float64 {
	factor := 5
	if gender == "female" {
		factor = -151
	}

	// Convert lbs to kgs.
	weight = weight * 0.45359237

	var bmr float64
	bmr = (10 * weight) + (6.25 * height) - (5 * float64(age)) + float64(factor)
	return bmr
}

// TDEE calcuates the Total Daily Energy Expenditure (TDEE) based on the
// BMR and user's activity level.
func TDEE(bmr float64, a string) float64 {
	al, err := activity(a)
	if err != nil {
		fmt.Println("Error:", err)
		return -1
	}

	return bmr * al
}

// Macros calculates and returns the recommended macronutrients given
// user weight and desired fat percentage.
func Macros(weight, fatPercent float64) (float64, float64, float64) {
	protein := 1 * weight
	fats := fatPercent * weight
	remaining := (protein * 4) + (fats * 9)
	carbs := remaining / 4

	return protein, carbs, fats
}
