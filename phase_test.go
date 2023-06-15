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

func ExampleCountEntriesPerWeek() {
	u := UserInfo{}
	u.Weight = 180
	u.Height = 180
	u.Age = 30
	u.ActivityLevel = "light"
	bmr := Mifflin(&u)
	u.TDEE = TDEE(bmr, u.ActivityLevel)

	weight := dataframe.NewSeriesString("weight", nil,
		"180", "180.1", "180.2", "180.3", "180.3", "180.4", "180.5",
		"180.6", "180.5", "180.6", "180.7", "180.8", "180.0", "181",
		"181.1", "181.2", "181.3", "181.4", "181.5", "181.5", "181.5")

	calories := dataframe.NewSeriesString("calories", nil,
		"2400", "2400", "2400", "2400", "2400", "2400", "2400",
		"2300", "2300", "2300", "2300", "2300", "2300", "2300",
		"2200", "2200", "2200", "2200", "2200", "2200", "2200")

	date := dataframe.NewSeriesString("date", nil,
		"2023-01-05", "2023-01-06", "2023-01-07", "2023-01-08", "2023-01-09", "2023-01-10", "2023-01-11",
		"2023-01-12", "2023-01-13", "2023-01-14", "2023-01-15", "2023-01-16", "2023-01-17", "2023-01-18",
		"2023-01-19", "2023-01-20", "2023-01-21", "2023-01-22", "2023-01-23", "2023-01-24", "2023-01-25")

	logs := dataframe.NewDataFrame(weight, calories, date)

	u.Phase.StartDate = time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)
	u.Phase.EndDate = time.Date(2023, time.January, 25, 0, 0, 0, 0, time.UTC)

	entryCountPerWeek, err := countEntriesPerWeek(&u, logs)

	for _, entries := range *entryCountPerWeek {
		fmt.Println(entries)
	}
	fmt.Println(err)

	// Output:
	// 7
	// 7
	// 7
	// <nil>
}

func ExampleCountEntriesInWeek() {

	weight := dataframe.NewSeriesString("weight", nil,
		"180", "182", "183", "184", "185")

	calories := dataframe.NewSeriesString("calories", nil,
		"2400", "2400", "2400", "2400", "2400")

	date := dataframe.NewSeriesString("date", nil,
		"2023-01-05", "2023-01-07", "2023-01-08", "2023-01-09", "2023-01-10")

	logs := dataframe.NewDataFrame(weight, calories, date)

	start := time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)
	end := time.Date(2023, time.January, 11, 0, 0, 0, 0, time.UTC)

	c, err := countEntriesInWeek(logs, start, end)

	fmt.Println(c)
	fmt.Println(err)

	// Output:
	// 5
	// <nil>
}

func ExampleCountValidWeeks() {
	m := make(map[int]int)
	m[1] = 6
	m[2] = 5
	m[3] = 7

	fmt.Println(countValidWeeks(m))

	// Output
	// 3
}

func ExampleRemoveCals() {
	u := UserInfo{}
	u.Weight = 180
	u.Height = 180
	u.Age = 30
	u.ActivityLevel = "light"
	bmr := Mifflin(&u)
	u.TDEE = TDEE(bmr, u.ActivityLevel)

	u.Phase.StartDate = time.Date(2023, time.January, 06, 0, 0, 0, 0, time.UTC)
	u.Phase.Duration = 8
	u.Phase.EndDate = calculateEndDate(u.Phase.StartDate, u.Phase.Duration)
	u.Phase.WeeklyChange = 0.75 // Desired weekly change in weight in pounds.
	u.Phase.GoalCalories = u.TDEE + (u.Phase.WeeklyChange * 500)
	u.Phase.LastCheckedDate = u.Phase.StartDate
	setMinMaxMacros(&u)
	u.Macros.Protein, u.Macros.Carbs, u.Macros.Fats = calculateMacros(&u)

	avgWeekWeightChange := 1.0 // User is gaining too much weight.

	removeCals(&u, avgWeekWeightChange)

	// Output:
	// Reducing caloric deficit by 125.00 calories.
	// New calorie goal: 2720.14.
}

func ExampleValidateAction() {
	err := validateAction("1")
	fmt.Println(err)

	// Output:
	// <nil>
}

func ExampleValidateNextAction() {
	err := validateNextAction("1")
	fmt.Println(err)

	// Output:
	// <nil>
}

func ExampleCheckCutLoss_withinRange() {
	u := UserInfo{}

	weight := dataframe.NewSeriesString("weight", nil,
		"181.1", "181.2", "181.3", "181.4", "181.5", "181.5", "181.5", // Within range
		"180.6", "180.5", "180.6", "180.7", "180.8", "180.0", "180.1", // Lost too much.
		"180", "180.1", "180.2", "180.3", "180.3", "180.4", "180.5") // Within range

	calories := dataframe.NewSeriesString("calories", nil,
		"2400", "2400", "2400", "2400", "2400", "2400", "2400",
		"2300", "2300", "2300", "2300", "2300", "2300", "2300",
		"2200", "2200", "2200", "2200", "2200", "2200", "2200")

	date := dataframe.NewSeriesString("date", nil,
		"2023-01-05", "2023-01-06", "2023-01-07", "2023-01-08", "2023-01-09", "2023-01-10", "2023-01-11",
		"2023-01-12", "2023-01-13", "2023-01-14", "2023-01-15", "2023-01-16", "2023-01-17", "2023-01-18",
		"2023-01-19", "2023-01-20", "2023-01-21", "2023-01-22", "2023-01-23", "2023-01-24", "2023-01-25")

	logs := dataframe.NewDataFrame(weight, calories, date)

	u.Phase.StartDate = time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)
	u.Phase.LastCheckedDate = u.Phase.StartDate
	u.Phase.EndDate = time.Date(2023, time.January, 25, 0, 0, 0, 0, time.UTC)
	u.Phase.WeeklyChange = -0.5

	status, avgTotal, err := checkCutLoss(&u, logs)

	fmt.Println(status)
	fmt.Println(avgTotal)
	fmt.Println(err)

	// Output:
	// 0
	// 0
	// <nil>
}

func ExampleCheckCutLoss_tooLittle() {
	u := UserInfo{}

	weight := dataframe.NewSeriesString("weight", nil,
		"180.4", "180.3", "180.3", "180.5", "180.2", "180.1", "180.1", // Lost too little.
		"180.1", "180", "179.9", "179.9", "180", "179.8", "179.8", // Lost too little.
		"179.5", "179.4", "179.4", "179.3", "179.2", "179.2", "179")

	calories := dataframe.NewSeriesString("calories", nil,
		"2400", "2400", "2400", "2400", "2400", "2400", "2400",
		"2300", "2300", "2300", "2300", "2300", "2300", "2300",
		"2200", "2200", "2200", "2200", "2200", "2200", "2200")

	date := dataframe.NewSeriesString("date", nil,
		"2023-01-05", "2023-01-06", "2023-01-07", "2023-01-08", "2023-01-09", "2023-01-10", "2023-01-11",
		"2023-01-12", "2023-01-13", "2023-01-14", "2023-01-15", "2023-01-16", "2023-01-17", "2023-01-18",
		"2023-01-19", "2023-01-20", "2023-01-21", "2023-01-22", "2023-01-23", "2023-01-24", "2023-01-25")

	logs := dataframe.NewDataFrame(weight, calories, date)

	u.Phase.StartDate = time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)
	u.Phase.LastCheckedDate = u.Phase.StartDate
	u.Phase.EndDate = time.Date(2023, time.January, 25, 0, 0, 0, 0, time.UTC)
	u.Phase.WeeklyChange = -0.5

	status, avgTotal, err := checkCutLoss(&u, logs)

	fmt.Println(status)
	fmt.Println(avgTotal)
	fmt.Println(err)

	// Output:
	// -1
	// -0.5999999999999943
	// <nil>
}

func ExampleCheckCutLoss_tooMuch() {
	u := UserInfo{}

	weight := dataframe.NewSeriesString("weight", nil,
		"171.8", "171.6", "171.4", "171.4", "171.4", "171.2", "171.0", // Lost too    much.
		"171", "170.8", "170.6", "170.6", "170.4", "170.4", "170.2", // Lost too    much.
		"170", "170", "170", "170", "170", "170", "170")

	calories := dataframe.NewSeriesString("calories", nil,
		"2400", "2400", "2400", "2400", "2400", "2400", "2400",
		"2300", "2300", "2300", "2300", "2300", "2300", "2300",
		"2200", "2200", "2200", "2200", "2200", "2200", "2200")

	date := dataframe.NewSeriesString("date", nil,
		"2023-01-05", "2023-01-06", "2023-01-07", "2023-01-08", "2023-01-09", "2023-01-10", "2023-01-11",
		"2023-01-12", "2023-01-13", "2023-01-14", "2023-01-15", "2023-01-16", "2023-01-17", "2023-01-18",
		"2023-01-19", "2023-01-20", "2023-01-21", "2023-01-22", "2023-01-23", "2023-01-24", "2023-01-25")

	logs := dataframe.NewDataFrame(weight, calories, date)

	u.Phase.StartDate = time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)
	u.Phase.LastCheckedDate = u.Phase.StartDate
	u.Phase.EndDate = time.Date(2023, time.January, 25, 0, 0, 0, 0, time.UTC)
	u.Phase.WeeklyChange = 0.5

	status, total, err := checkCutLoss(&u, logs)

	fmt.Println(status)
	fmt.Println(total)
	fmt.Println(err)

	// Output:
	// 1
	// -1.6000000000000227
	// <nil>
}

func ExampleMetWeeklyGoalCut() {
	u := UserInfo{}
	u.Phase.WeeklyChange = -0.5
	status := metWeeklyGoalCut(&u, -0.45) // Did not lose enough weight
	fmt.Println(status)

	// Output:
	// 0
}

func ExampleCheckMaintenance() {
	u := UserInfo{}
	u.Phase.StartDate = time.Date(2023, time.January, 06, 0, 0, 0, 0, time.UTC)

	weight := dataframe.NewSeriesString("weight", nil,
		"180.3", "180.1", "180.1", "180.2", "180.15", "180.22", "180.42", // Maintained
		"180.4", "180.1", "180.2", "180.2", "180.4", "180.39", "180.4", // Maintained
		"180.3", "180.2", "180.2", "180.2", "180.1", "180.3", "180.1") // Maintained

	calories := dataframe.NewSeriesString("calories", nil,
		"2400", "2400", "2400", "2400", "2400", "2400", "2400",
		"2300", "2300", "2300", "2300", "2300", "2300", "2300",
		"2200", "2200", "2200", "2200", "2200", "2200", "2200")

	date := dataframe.NewSeriesString("date", nil,
		"2023-01-05", "2023-01-06", "2023-01-07", "2023-01-08", "2023-01-09", "2023-01-10", "2023-01-11",
		"2023-01-12", "2023-01-13", "2023-01-14", "2023-01-15", "2023-01-16", "2023-01-17", "2023-01-18",
		"2023-01-19", "2023-01-20", "2023-01-21", "2023-01-22", "2023-01-23", "2023-01-24", "2023-01-25")

	logs := dataframe.NewDataFrame(weight, calories, date)

	u.Phase.WeeklyChange = 0
	u.Phase.StartDate = time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)
	u.Phase.LastCheckedDate = u.Phase.StartDate
	u.Phase.EndDate = time.Date(2023, time.January, 25, 0, 0, 0, 0, time.UTC)

	status, total, err := checkMaintenance(&u, logs)

	fmt.Println(status)
	fmt.Println(total)
	fmt.Println(err)

	// Output:
	// 0
	// 0
	// <nil>
}

func ExampleMetWeeklyGoalMaintenance() {
	u := UserInfo{}
	u.Phase.WeeklyChange = 0
	status := metWeeklyGoalMainenance(&u, 0.05) // Within range.
	fmt.Println(status)

	// Output:
	// 0
}

func ExampleCheckBulkGain_withinRange() {
	u := UserInfo{}

	weight := dataframe.NewSeriesString("weight", nil,
		"180", "180.1", "180.2", "180.3", "180.3", "180.4", "180.5",
		"180.6", "180.5", "180.6", "180.7", "180.8", "180.0", "181",
		"181.1", "181.2", "181.3", "181.4", "181.5", "181.5", "181.5")

	calories := dataframe.NewSeriesString("calories", nil,
		"2400", "2400", "2400", "2400", "2400", "2400", "2400",
		"2300", "2300", "2300", "2300", "2300", "2300", "2300",
		"2200", "2200", "2200", "2200", "2200", "2200", "2200")

	date := dataframe.NewSeriesString("date", nil,
		"2023-01-05", "2023-01-06", "2023-01-07", "2023-01-08", "2023-01-09", "2023-01-10", "2023-01-11",
		"2023-01-12", "2023-01-13", "2023-01-14", "2023-01-15", "2023-01-16", "2023-01-17", "2023-01-18",
		"2023-01-19", "2023-01-20", "2023-01-21", "2023-01-22", "2023-01-23", "2023-01-24", "2023-01-25")

	logs := dataframe.NewDataFrame(weight, calories, date)

	u.Phase.StartDate = time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)
	u.Phase.LastCheckedDate = u.Phase.StartDate
	u.Phase.EndDate = time.Date(2023, time.January, 25, 0, 0, 0, 0, time.UTC)
	u.Phase.WeeklyChange = 0.5

	status, avgTotal, err := checkBulkGain(&u, logs)

	fmt.Println(status)
	fmt.Println(avgTotal)
	fmt.Println(err)

	// Output:
	// 0
	// 0
	// <nil>
}

func ExampleCheckBulkGain_tooLittle() {
	u := UserInfo{}

	weight := dataframe.NewSeriesString("weight", nil,
		"180", "180", "180", "180", "180", "180", "180", // Gained too little.
		"175", "175", "175", "175", "175", "175", "175", // Gained too little.
		"170", "170", "170", "170", "170", "170", "170")

	calories := dataframe.NewSeriesString("calories", nil,
		"2400", "2400", "2400", "2400", "2400", "2400", "2400",
		"2300", "2300", "2300", "2300", "2300", "2300", "2300",
		"2200", "2200", "2200", "2200", "2200", "2200", "2200")

	date := dataframe.NewSeriesString("date", nil,
		"2023-01-05", "2023-01-06", "2023-01-07", "2023-01-08", "2023-01-09", "2023-01-10", "2023-01-11",
		"2023-01-12", "2023-01-13", "2023-01-14", "2023-01-15", "2023-01-16", "2023-01-17", "2023-01-18",
		"2023-01-19", "2023-01-20", "2023-01-21", "2023-01-22", "2023-01-23", "2023-01-24", "2023-01-25")

	logs := dataframe.NewDataFrame(weight, calories, date)

	u.Phase.StartDate = time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)
	u.Phase.LastCheckedDate = u.Phase.StartDate
	u.Phase.EndDate = time.Date(2023, time.January, 25, 0, 0, 0, 0, time.UTC)
	u.Phase.WeeklyChange = 0.5

	status, avgTotal, err := checkBulkGain(&u, logs)

	fmt.Println(status)
	fmt.Println(avgTotal)
	fmt.Println(err)

	// Output:
	// -1
	// -5
	// <nil>
}

func ExampleCheckBulkGain_tooMuch() {
	u := UserInfo{}

	weight := dataframe.NewSeriesString("weight", nil,
		"170", "170", "170", "170", "170", "170", "170", // Gained too little.
		"170.2", "170.4", "170.4", "170.4", "170.6", "170.8", "170.8", // Gained too much.
		"171.0", "171.2", "171.2", "171.4", "171.4", "171.6", "171.8") // Gained too much.

	calories := dataframe.NewSeriesString("calories", nil,
		"2400", "2400", "2400", "2400", "2400", "2400", "2400",
		"2300", "2300", "2300", "2300", "2300", "2300", "2300",
		"2200", "2200", "2200", "2200", "2200", "2200", "2200")

	date := dataframe.NewSeriesString("date", nil,
		"2023-01-05", "2023-01-06", "2023-01-07", "2023-01-08", "2023-01-09", "2023-01-10", "2023-01-11",
		"2023-01-12", "2023-01-13", "2023-01-14", "2023-01-15", "2023-01-16", "2023-01-17", "2023-01-18",
		"2023-01-19", "2023-01-20", "2023-01-21", "2023-01-22", "2023-01-23", "2023-01-24", "2023-01-25")

	logs := dataframe.NewDataFrame(weight, calories, date)

	u.Phase.StartDate = time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)
	u.Phase.LastCheckedDate = u.Phase.StartDate
	u.Phase.EndDate = time.Date(2023, time.January, 25, 0, 0, 0, 0, time.UTC)
	u.Phase.WeeklyChange = 0.5

	status, total, err := checkBulkGain(&u, logs)

	fmt.Println(status)
	fmt.Println(total)
	fmt.Println(err)

	// Output:
	// 1
	// 1.8000000000000114
	// <nil>
}

func ExampleMetWeeklyGoalBulk() {
	u := UserInfo{}
	u.Phase.WeeklyChange = 0.5
	status := metWeeklyGoalBulk(&u, 0.3) // gained too little
	fmt.Println(status)

	// Output:
	// -1
}

func ExampleAddCals() {
	u := UserInfo{}
	u.Weight = 180
	u.Height = 180
	u.Age = 30
	u.ActivityLevel = "light"
	bmr := Mifflin(&u)
	u.TDEE = TDEE(bmr, u.ActivityLevel)

	u.Phase.StartDate = time.Date(2023, time.January, 06, 0, 0, 0, 0, time.UTC)
	u.Phase.Duration = 8
	u.Phase.EndDate = calculateEndDate(u.Phase.StartDate, u.Phase.Duration)
	u.Phase.WeeklyChange = 0.75 // Desired weekly change in weight in pounds.
	u.Phase.GoalCalories = u.TDEE + (u.Phase.WeeklyChange * 500)
	u.Phase.LastCheckedDate = u.Phase.StartDate
	setMinMaxMacros(&u)
	u.Macros.Protein, u.Macros.Carbs, u.Macros.Fats = calculateMacros(&u)

	avgWeekWeightChange := 0.50 // User is not gaining enough weight.

	addCals(&u, avgWeekWeightChange)

	// Output:
	// Adding to caloric surplus by 125.00 calories.
	// New calorie goal: 2970.14.
}

func ExampleTotalWeightChangeWeek() {
	u := UserInfo{}
	u.Phase.StartDate = time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)

	weight := dataframe.NewSeriesString("weight", nil,
		"180", "181", "182", "183", "184", "185", "186")

	calories := dataframe.NewSeriesString("calories", nil,
		"2400", "2400", "2400", "2400", "2400", "2400", "2400")

	date := dataframe.NewSeriesString("date", nil,
		"2023-01-05", "2023-01-06", "2023-01-07", "2023-01-08", "2023-01-09", "2023-01-10", "2023-01-11")

	logs := dataframe.NewDataFrame(weight, calories, date)

	start := time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)
	end := time.Date(2023, time.January, 11, 0, 0, 0, 0, time.UTC)

	avg, _, _ := totalWeightChangeWeek(logs, start, end, &u)
	fmt.Println(avg)

	// Output:
	// 6
}

func ExampleFindEntryIdx() {
	weight := dataframe.NewSeriesString("weight", nil, "180", "182", "180.5", "181.1",
		"182.2", "182.1", "183.4", "183", "183.3", "183.2")
	calories := dataframe.NewSeriesString("calories", nil, "2410", "2490", "2573", "2400",
		"2408", "2499", "2550", "2570", "2600", "2599")
	date := dataframe.NewSeriesString("date", nil, "2023-01-05", "2023-01-06", "2023-01-07",
		"2023-01-08", "2023-01-09", "2023-01-10",
		"2023-01-11", "2023-01-12", "2023-01-13", "2023-01-14")
	logs := dataframe.NewDataFrame(weight, calories, date)
	day := time.Date(2023, time.January, 8, 0, 0, 0, 0, time.UTC)

	i, err := findEntryIdx(logs, day)

	fmt.Println(i)
	fmt.Println(err)

	// Output:
	// 3
	// <nil>
}

func ExampleGetPrecedingWeightToDay() {
	u := UserInfo{}
	u.Phase.StartDate = time.Date(2023, time.January, 06, 0, 0, 0, 0, time.UTC)

	weight := dataframe.NewSeriesString("weight", nil, "180", "182", "180.5")
	calories := dataframe.NewSeriesString("calories", nil, "2410", "2490", "2573")
	date := dataframe.NewSeriesString("date", nil, "2023-01-05", "2023-01-06", "2023-01-07")
	logs := dataframe.NewDataFrame(weight, calories, date)
	startIdx := 2 // Index of the succeeding date.

	w, err := getPrecedingWeightToDay(&u, logs, 180.5, startIdx)

	fmt.Println(w)
	fmt.Println(err)

	// Output:
	// 182
	// <nil>
}

func ExampleGetPrecedingWeightToDay_beforePhase() {
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
	// -1.25
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
