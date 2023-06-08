package calories

import (
	"fmt"
	"testing"
	"time"

	"github.com/rocketlaunchr/dataframe-go"
)

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

func ExampleValidateWeight() {
	w, err := validateWeight("180")
	fmt.Println(w)
	fmt.Println(err)

	// Output:
	// 180
	// <nil>
}

func ExampleValidateWeight_error() {
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

func ExampleValidateHeight_error() {
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

func ExampleGetAvgWeightChangeWeek() {
	u := UserInfo{}
	u.Phase.StartDate = time.Date(2023, time.January, 06, 0, 0, 0, 0, time.UTC)
	weight := dataframe.NewSeriesString("weight", nil, "180", "182", "180.5", "181.1",
		"182.2", "182.1", "183.4", "183", "183.3", "183.2")
	calories := dataframe.NewSeriesString("calories", nil, "2410", "2490", "2573", "2400",
		"2408", "2499", "2550", "2570", "2600", "2599")
	date := dataframe.NewSeriesString("date", nil, "2023-01-05", "2023-01-06", "2023-01-07",
		"2023-01-08", "2023-01-09", "2023-01-10",
		"2023-01-11", "2023-01-12", "2023-01-13", "2023-01-14")
	logs := dataframe.NewDataFrame(weight, calories, date)
	start := time.Date(2023, time.January, 6, 0, 0, 0, 0, time.UTC)
	end := time.Date(2023, time.January, 13, 0, 0, 0, 0, time.UTC)

	avg, _, _ := avgWeightChangeWeek(logs, start, end, &u)
	fmt.Println(avg)

	// Output:
	// 0.14285714285714285
}

func ExampleGetPrecedingWeightToDay() {
	u := UserInfo{}
	u.Phase.StartDate = time.Date(2023, time.January, 06, 0, 0, 0, 0, time.UTC)

	weight := dataframe.NewSeriesString("weight", nil, "180", "182", "180.5")
	calories := dataframe.NewSeriesString("calories", nil, "2410", "2490", "2573")
	date := dataframe.NewSeriesString("date", nil, "2023-01-05", "2023-01-06", "2023-01-07")
	logs := dataframe.NewDataFrame(weight, calories, date)
	startIdx := 1 // Index of the succeeding date.

	w, err := getPrecedingWeightToDay(&u, logs, 182, startIdx)

	fmt.Println(w)
	fmt.Println(err)

	// Output:
	// 182
	// <nil>
}

// TODO: func ExampleGetPrecedingWeightToDay_zeroIdx

func ExampleValidateActivity_error() {
	err := validateActivity("foo")
	fmt.Println(err)

	// Output:
	// unknown activity level: foo
}

func ExampleValidateDietChoice() {
	err := validateDietChoice("custom")
	fmt.Println(err)

	// Output:
	// <nil>
}

func ExampleSetRecommendedValues() {
	u := UserInfo{}
	u.Phase.StartDate = time.Date(2023, time.January, 01, 0, 0, 0, 0, time.UTC)
	setRecommendedValues(&u, 1.25, 8, 170, 2300)
	fmt.Println(u.Phase.WeeklyChange)
	fmt.Println(u.Phase.Duration)
	fmt.Println(u.Phase.GoalWeight)
	fmt.Println(u.Phase.GoalCalories)
	fmt.Println(u.Phase.LastCheckedDate)

	// Output:
	// 1.25
	// 8
	// 170
	// 2300
	// 2023-01-01 00:00:00 +0000 UTC
}

func ExampleCalculateEndDate() {
	start := time.Date(2023, time.January, 01, 0, 0, 0, 0, time.UTC)
	dietDuration := 2.3 // 2 weeks and 2 days.
	end := calculateEndDate(start, dietDuration)
	fmt.Println(end)

	// Output:
	// 2023-01-17 00:00:00 +0000 UTC
}

func ExampleValidateEndDate() {
	u := UserInfo{}
	u.Phase.StartDate = time.Date(2023, time.May, 20, 0, 0, 0, 0, time.UTC)
	u.Phase.MaxDuration = 12
	dateStr := "2023-08-05"

	d, dur, err := validateEndDate(dateStr, &u)
	fmt.Println(d)
	fmt.Println(dur)
	fmt.Println(err)

	// Output:
	// 2023-08-05 00:00:00 +0000 UTC
	// 11
	// <nil>
}

func ExampleValidateEndDate_error() {
	u := UserInfo{}
	u.Phase.StartDate = time.Date(2023, time.January, 01, 0, 0, 0, 0, time.UTC)
	u.Phase.MaxDuration = 12
	u.Phase.MinDuration = 6
	dateStr := "2023-04-01"

	d, dur, err := validateEndDate(dateStr, &u)
	fmt.Println(d)
	fmt.Println(dur)
	fmt.Println(err)

	// Output:
	// 0001-01-01 00:00:00 +0000 UTC
	// 0
	// Invalid diet phase end date. Diet duration of 12.86 weeks exceeds the maximum duration of 12.00.
}

func ExampleValidateDate() {
	dateStr := "2023-01-23"
	date, err := validateDate(dateStr)
	fmt.Println(date)
	fmt.Println(err)

	// Output:
	// 2023-01-23 00:00:00 +0000 UTC
	// <nil>
}

func TestValidateDate_parseError(t *testing.T) {
	dateStr := "2023 01 23"
	_, err := validateDate(dateStr)

	if err == nil {
		t.Error("Expected error, but got nil")
	}
}

func ExampleCalculateDuration() {
	year := 2023
	month := time.May
	day := 20

	start := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	end := time.Date(year, month+4, day, 0, 0, 0, 0, time.UTC)
	dur := calculateDuration(start, end)
	fmt.Println(dur)

	// Output:
	// 2952h0m0s
}

func ExampleValidateGoalWeight_cut() {
	weightStr := "180"
	u := UserInfo{}
	u.Phase.Name = "cut"
	u.Phase.StartWeight = 190
	g, err := validateGoalWeight(weightStr, &u)

	fmt.Println(g)
	fmt.Println(err)

	// Output:
	// 180
	// <nil>
}

func ExampleValidateGoalWeight_invalidInput() {
	weightStr := "foo"
	u := UserInfo{}
	u.Phase.Name = "cut"
	u.Phase.StartWeight = 190
	g, err := validateGoalWeight(weightStr, &u)

	fmt.Println(g)
	fmt.Println(err)

	// Output:
	// 0
	// Invalid goal weight. Goal weight must be a number.
}

func ExampleValidateGoalWeight_invalidCut() {
	weightStr := "150"
	u := UserInfo{}
	u.Phase.Name = "cut"
	u.Phase.StartWeight = 190
	g, err := validateGoalWeight(weightStr, &u)

	fmt.Println(g)
	fmt.Println(err)

	// Output:
	// 0
	// Invalid goal weight. For a cut, goal weight cannot be less than 10% of starting body weight.
}

func ExampleValidateGoalWeight_invalidBulk() {
	weightStr := "210"
	u := UserInfo{}
	u.Phase.Name = "bulk"
	u.Phase.StartWeight = 190
	g, err := validateGoalWeight(weightStr, &u)

	fmt.Println(g)
	fmt.Println(err)

	// Output:
	// 0
	// Invalid goal weight. For a bulk, goal weight cannot exceed 10% of starting body weight.
}

func ExampleCalculateWeeklyChange_cut() {
	curWeight := 180.0 // Current weight
	goalWeight := 170.0
	dur := 8.0 // Diet duration
	weeklyChange := calculateWeeklyChange(curWeight, goalWeight, dur)
	fmt.Println(weeklyChange)

	// Output:
	// 1.25
}

func ExampleCalculateWeeklyChange_bulk() {
	curWeight := 180.0 // Current weight
	goalWeight := 210.0
	dur := 8.0 // Diet duration
	weeklyChange := calculateWeeklyChange(curWeight, goalWeight, dur)
	fmt.Println(weeklyChange)

	// Output:
	// 3.75
}

func ExampleSetMinMaxPhaseDuration() {
	u := UserInfo{}
	u.Phase.Name = "cut"
	setMinMaxPhaseDuration(&u)

	fmt.Println(u.Phase.MaxDuration)
	fmt.Println(u.Phase.MinDuration)

	// Output:
	// 12
	// 6
}

func ExampleSetMinMaxPhaseDuration_error() {
	u := UserInfo{}
	u.Phase.Name = "foo"
	setMinMaxPhaseDuration(&u)

	fmt.Println(u.Phase.MaxDuration)
	fmt.Println(u.Phase.MinDuration)

	// Output:
	// 0
	// 0
}

func ExampleValidateDietPhase() {
	err := validateDietPhase("maintain")
	fmt.Println(err)

	// Output:
	// <nil>
}

func ExampleValidateDietPhase_error() {
	err := validateDietPhase("foo")
	fmt.Println(err)

	// Output:
	// Invalid diet phase.
}
