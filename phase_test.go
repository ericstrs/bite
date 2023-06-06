package calories

import "fmt"

func ExampleValidateSex() {
	err := validateSex("male")
	fmt.Println(err)

	// Output:
	// <nil>
}

func ExampleValidateSexError() {
	err := validateSex("foo")
	fmt.Println(err)

	// Output:
	// Invalid sex.
}

func ExampleValidateWeight() {
	w, err := validateWeight("180")
	fmt.Println(w)
	fmt.Println(err)

	// Output:
	// 180
	// <nil>
}

func ExampleValidateWeightError() {
	w, err := validateWeight("foo")
	fmt.Println(w)
	fmt.Println(err)

	// Output:
	// 0
	// Invalid weight.
}

func ExampleValidateHeight() {
	h, err := validateHeight("170.0")
	fmt.Println(h)
	fmt.Println(err)

	// Output:
	// 170
	// <nil>
}

func ExampleValidateHeightError() {
	h, err := validateHeight("foo")
	fmt.Println(h)
	fmt.Println(err)

	// Output:
	// 0
	// Invalid height.
}

func ExampleValidateAge() {
	a, err := validateAge("30")
	fmt.Println(a)
	fmt.Println(err)

	// Output:
	// 30
	// <nil>
}

func ExampleValidateAgeError() {
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

func ExampleValidateActivityError() {
	err := validateActivity("foo")
	fmt.Println(err)

	// Output:
	// unknown activity level: foo
}
