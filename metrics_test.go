package calories_test

import (
	"fmt"

	m "github.com/oneseIf/calories"
)

func ExampleMifflin() {
	weight := 80.0  // kg
	height := 180.0 // cm
	age := 30
	gender := "male"
	result := m.Mifflin(weight, height, age, gender)
	fmt.Println(result)

	// Output:
	// 1780
}
