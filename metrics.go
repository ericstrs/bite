// Calories package provides dietary insight.
package calories

// Mifflin calculates the Basal Metabolic Rate (BMR) which is based on
// weight (kg), height (cm), age (years), and gender.
//
// It returns BMR.
func Mifflin(weight, height float64, age int, gender string) float64 {

	factor := 5
	if gender == "female" {
		factor = -151
	}

	var bmr float64
	bmr = (10 * weight) + (6.25 * height) - (5 * float64(age)) + float64(factor)
	return bmr
}
