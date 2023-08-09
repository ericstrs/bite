package calories

import (
	"fmt"
	"log"
	"sort"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

func ExampleCountEntriesPerWeek() {
	u := UserInfo{}
	u.Weight = 180
	u.Height = 180
	u.Age = 30
	u.ActivityLevel = "light"
	bmr := Mifflin(&u)
	u.TDEE = TDEE(bmr, u.ActivityLevel)

	entries := []Entry{
		{UserWeight: 180.0, Calories: 2400, Date: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.1, Calories: 2400, Date: time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.2, Calories: 2400, Date: time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.3, Calories: 2400, Date: time.Date(2023, 1, 8, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.3, Calories: 2400, Date: time.Date(2023, 1, 9, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.4, Calories: 2400, Date: time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.5, Calories: 2400, Date: time.Date(2023, 1, 11, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.6, Calories: 2300, Date: time.Date(2023, 1, 12, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.5, Calories: 2300, Date: time.Date(2023, 1, 13, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.6, Calories: 2300, Date: time.Date(2023, 1, 14, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.7, Calories: 2300, Date: time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.8, Calories: 2300, Date: time.Date(2023, 1, 16, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.0, Calories: 2300, Date: time.Date(2023, 1, 17, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.0, Calories: 2300, Date: time.Date(2023, 1, 18, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.1, Calories: 2200, Date: time.Date(2023, 1, 19, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.2, Calories: 2200, Date: time.Date(2023, 1, 20, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.3, Calories: 2200, Date: time.Date(2023, 1, 21, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.4, Calories: 2200, Date: time.Date(2023, 1, 22, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.5, Calories: 2200, Date: time.Date(2023, 1, 23, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.5, Calories: 2200, Date: time.Date(2023, 1, 24, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.5, Calories: 2200, Date: time.Date(2023, 1, 25, 0, 0, 0, 0, time.UTC)},
	}

	/*
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
	*/

	u.Phase.StartDate = time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)
	u.Phase.EndDate = time.Date(2023, time.January, 25, 0, 0, 0, 0, time.UTC)

	entryCountPerWeek, err := countEntriesPerWeek(&u, &entries)

	// Must sort the keys since iteration over maps is not guaranteed.

	// Get the keys and sort them.
	weeks := make([]int, 0, len(*entryCountPerWeek))
	for week := range *entryCountPerWeek {
		weeks = append(weeks, week)
	}
	sort.Ints(weeks)

	// Print the entries in the order of the sorted keys.
	for _, week := range weeks {
		entries := (*entryCountPerWeek)[week]
		fmt.Printf("Week %d entries: %d\n", week, entries)
	}

	/*
		for week, entries := range *entryCountPerWeek {
			fmt.Printf("Week %d entries: %d\n", week, entries)
		}
	*/
	fmt.Println(err)

	// Output:
	// Week 0 entries: 4
	// Week 1 entries: 7
	// Week 2 entries: 7
	// Week 3 entries: 3
	// <nil>
}

func ExampleCountEntriesInWeek() {
	/*
		weight := dataframe.NewSeriesString("weight", nil,
			"180", "182", "183", "184", "185")

		calories := dataframe.NewSeriesString("calories", nil,
			"2400", "2400", "2400", "2400", "2400")

		date := dataframe.NewSeriesString("date", nil,
			"2023-01-05", "2023-01-07", "2023-01-08", "2023-01-09", "2023-01-10")

		logs := dataframe.NewDataFrame(weight, calories, date)
	*/

	entries := []Entry{
		{UserWeight: 180.0, Calories: 2400, Date: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 182.0, Calories: 2400, Date: time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 183.0, Calories: 2400, Date: time.Date(2023, 1, 8, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 184.0, Calories: 2400, Date: time.Date(2023, 1, 9, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 185.0, Calories: 2400, Date: time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)},
	}

	start := time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)
	end := time.Date(2023, time.January, 11, 0, 0, 0, 0, time.UTC)

	c, err := countEntriesInWeek(&entries, start, end)

	fmt.Println(c)
	fmt.Println(err)

	// Output:
	// 5
	// <nil>
}

func ExampleCountEntriesInWeek_startDate() {
	/*
		weight := dataframe.NewSeriesString("weight", nil,
			"180", "182", "183", "184", "185")

		calories := dataframe.NewSeriesString("calories", nil,
			"2400", "2400", "2400", "2400", "2400")

		date := dataframe.NewSeriesString("date", nil,
			"2023-01-05", "2023-01-07", "2023-01-08", "2023-01-09", "2023-01-10")

		logs := dataframe.NewDataFrame(weight, calories, date)
	*/

	entries := []Entry{
		{UserWeight: 180.0, Calories: 2400, Date: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 182.0, Calories: 2400, Date: time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 183.0, Calories: 2400, Date: time.Date(2023, 1, 8, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 184.0, Calories: 2400, Date: time.Date(2023, 1, 9, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 185.0, Calories: 2400, Date: time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)},
	}

	start := time.Date(2023, time.January, 4, 0, 0, 0, 0, time.UTC)
	end := time.Date(2023, time.January, 11, 0, 0, 0, 0, time.UTC)

	c, err := countEntriesInWeek(&entries, start, end)

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
	u.Weight = 180 // lbs
	u.Height = 70  // inches
	u.Age = 30
	u.ActivityLevel = "light"
	bmr := Mifflin(&u)
	u.TDEE = TDEE(bmr, u.ActivityLevel)

	u.Phase.StartDate = time.Date(2023, time.January, 06, 0, 0, 0, 0, time.UTC)
	u.Phase.Duration = 8
	u.Phase.EndDate = calculateEndDate(u.Phase.StartDate, u.Phase.Duration)
	u.Phase.WeeklyChange = 0.75 // Desired weekly change in weight in pounds.
	u.Phase.GoalCalories = u.TDEE + (u.Phase.WeeklyChange * 500)
	u.Phase.LastCheckedWeek = u.Phase.StartDate
	setMinMaxMacros(&u)
	u.Macros.Protein, u.Macros.Carbs, u.Macros.Fats = calculateMacros(&u)

	avgWeekWeightChange := 1.0 // User is gaining too much weight.

	removeCals(&u, avgWeekWeightChange)

	// Output:
	// Reducing caloric deficit by 125.00 calories.
	// New calorie goal: 2701.23.
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

	/*
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
	*/

	entries := []Entry{
		{UserWeight: 181.1, Calories: 2400, Date: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.2, Calories: 2400, Date: time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.3, Calories: 2400, Date: time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.4, Calories: 2400, Date: time.Date(2023, 1, 8, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.5, Calories: 2400, Date: time.Date(2023, 1, 9, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.5, Calories: 2400, Date: time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.5, Calories: 2400, Date: time.Date(2023, 1, 11, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.6, Calories: 2300, Date: time.Date(2023, 1, 12, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.5, Calories: 2300, Date: time.Date(2023, 1, 13, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.6, Calories: 2300, Date: time.Date(2023, 1, 14, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.7, Calories: 2300, Date: time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.8, Calories: 2300, Date: time.Date(2023, 1, 16, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.0, Calories: 2300, Date: time.Date(2023, 1, 17, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.1, Calories: 2300, Date: time.Date(2023, 1, 18, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.0, Calories: 2200, Date: time.Date(2023, 1, 19, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.1, Calories: 2200, Date: time.Date(2023, 1, 20, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.2, Calories: 2200, Date: time.Date(2023, 1, 21, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.3, Calories: 2200, Date: time.Date(2023, 1, 22, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.3, Calories: 2200, Date: time.Date(2023, 1, 23, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.4, Calories: 2200, Date: time.Date(2023, 1, 24, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.5, Calories: 2200, Date: time.Date(2023, 1, 25, 0, 0, 0, 0, time.UTC)},
	}

	u.Phase.StartDate = time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)
	u.Phase.LastCheckedWeek = u.Phase.StartDate
	u.Phase.EndDate = time.Date(2023, time.January, 25, 0, 0, 0, 0, time.UTC)
	u.Phase.WeeklyChange = -0.5
	u.Phase.GoalCalories = 2400
	u.Phase.Status = "active"
	u.Phase.Name = "cut"

	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Start a new transaction.
	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
		return
	}

	err = setupTestConfigTables(tx)
	if err != nil {
		return
	}

	/*
	  // Insert dummy data
	  _, err = db.Exec(`
	    INSERT INTO macros (protein, min_protein, max_protein, carbs, min_carbs, max_carbs, fats, min_fats, max_fats)
	    VALUES (100, 90, 110, 200, 180, 220, 50, 45, 55);

	    INSERT INTO phase_info (user_id, name, goal_calories, start_weight, goal_weight, weight_change_threshold, weekly_change, start_date, end_date, last_checked_week, duration, max_duration, min_duration, active)
	    VALUES (1, 'Weight Loss', 2000, 190, 170, 2, -1, '2023-01-01', '2023-04-01', '2023-01-07', 12, 16, 8, 1);

	    INSERT INTO config (sex, weight, height, age, activity_level, tdee, system, macros_id, phase_id)
	    VALUES ('M', 190, 175, 30, 'Moderate', 2500, 'Imperial', 1, 1);
	    `)
	  if err != nil {
	    log.Printf("Failed to insert dummy data: %v", err)
	    return
	  }
	*/

	status, avgTotal, err := checkCutLoss(tx, &u, &entries)

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

	/*
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
	*/

	entries := []Entry{
		{UserWeight: 180.4, Calories: 2400, Date: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.3, Calories: 2400, Date: time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.3, Calories: 2400, Date: time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.5, Calories: 2400, Date: time.Date(2023, 1, 8, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.2, Calories: 2400, Date: time.Date(2023, 1, 9, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.1, Calories: 2400, Date: time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.1, Calories: 2400, Date: time.Date(2023, 1, 11, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.1, Calories: 2300, Date: time.Date(2023, 1, 12, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.0, Calories: 2300, Date: time.Date(2023, 1, 13, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 179.9, Calories: 2300, Date: time.Date(2023, 1, 14, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 179.9, Calories: 2300, Date: time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.0, Calories: 2300, Date: time.Date(2023, 1, 16, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 179.8, Calories: 2300, Date: time.Date(2023, 1, 17, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 179.8, Calories: 2300, Date: time.Date(2023, 1, 18, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 179.5, Calories: 2200, Date: time.Date(2023, 1, 19, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 179.4, Calories: 2200, Date: time.Date(2023, 1, 20, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 179.4, Calories: 2200, Date: time.Date(2023, 1, 21, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 179.3, Calories: 2200, Date: time.Date(2023, 1, 22, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 179.2, Calories: 2200, Date: time.Date(2023, 1, 23, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 179.2, Calories: 2200, Date: time.Date(2023, 1, 24, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 179.0, Calories: 2200, Date: time.Date(2023, 1, 25, 0, 0, 0, 0, time.UTC)},
	}

	u.Phase.StartDate = time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)
	u.Phase.LastCheckedWeek = u.Phase.StartDate
	u.Phase.EndDate = time.Date(2023, time.January, 25, 0, 0, 0, 0, time.UTC)
	u.Phase.WeeklyChange = -0.5
	u.Phase.GoalCalories = 2400
	u.Phase.Name = "cut"
	u.Phase.Status = "active"

	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Start a new transaction.
	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
		return
	}

	err = setupTestConfigTables(tx)
	if err != nil {
		return
	}

	status, avgTotal, err := checkCutLoss(tx, &u, &entries)

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

	/*
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
	*/

	entries := []Entry{
		{UserWeight: 171.8, Calories: 2400, Date: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 171.6, Calories: 2400, Date: time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 171.4, Calories: 2400, Date: time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 171.4, Calories: 2400, Date: time.Date(2023, 1, 8, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 171.4, Calories: 2400, Date: time.Date(2023, 1, 9, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 171.2, Calories: 2400, Date: time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 171.0, Calories: 2400, Date: time.Date(2023, 1, 11, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 171.0, Calories: 2300, Date: time.Date(2023, 1, 12, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.8, Calories: 2300, Date: time.Date(2023, 1, 13, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.6, Calories: 2300, Date: time.Date(2023, 1, 14, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.6, Calories: 2300, Date: time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.4, Calories: 2300, Date: time.Date(2023, 1, 16, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.4, Calories: 2300, Date: time.Date(2023, 1, 17, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.2, Calories: 2300, Date: time.Date(2023, 1, 18, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.0, Calories: 2200, Date: time.Date(2023, 1, 19, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.0, Calories: 2200, Date: time.Date(2023, 1, 20, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.0, Calories: 2200, Date: time.Date(2023, 1, 21, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.0, Calories: 2200, Date: time.Date(2023, 1, 22, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.0, Calories: 2200, Date: time.Date(2023, 1, 23, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.0, Calories: 2200, Date: time.Date(2023, 1, 24, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.0, Calories: 2200, Date: time.Date(2023, 1, 25, 0, 0, 0, 0, time.UTC)},
	}

	u.Phase.StartDate = time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)
	u.Phase.LastCheckedWeek = u.Phase.StartDate
	u.Phase.EndDate = time.Date(2023, time.January, 25, 0, 0, 0, 0, time.UTC)
	u.Phase.WeeklyChange = 0.5
	u.Phase.GoalCalories = 2400
	u.Phase.Name = "cut"
	u.Phase.Status = "active"

	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Start a new transaction.
	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
		return
	}

	err = setupTestConfigTables(tx)
	if err != nil {
		return
	}

	status, total, err := checkCutLoss(tx, &u, &entries)

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

func ExampleCheckMaintenance_within() {
	u := UserInfo{}
	u.Phase.StartDate = time.Date(2023, time.January, 06, 0, 0, 0, 0, time.UTC)

	/*
		weight := dataframe.NewSeriesString("weight", nil,
			"180.3", "180.1", "180.1", "180.2", "180.15", "180.22", "180.42", // Maintained
			"180.4", "180.1", "180.2", "180.2", "180.4", "180.39", "180.4", // Maintained
			"180.3", "180.2", "180.2", "180.2", "180.1", "180.3", "180.1") // Maintained

		calories := dataframe.NewSeriesString("calories", nil,
			"2400", "2400", "2400", "2400", "2400", "2400", "2400",
			"2400", "2400", "2400", "2400", "2400", "2400", "2400",
			"2400", "2400", "2400", "2400", "2400", "2400", "2400")

		date := dataframe.NewSeriesString("date", nil,
			"2023-01-05", "2023-01-06", "2023-01-07", "2023-01-08", "2023-01-09", "2023-01-10", "2023-01-11",
			"2023-01-12", "2023-01-13", "2023-01-14", "2023-01-15", "2023-01-16", "2023-01-17", "2023-01-18",
			"2023-01-19", "2023-01-20", "2023-01-21", "2023-01-22", "2023-01-23", "2023-01-24", "2023-01-25")

		logs := dataframe.NewDataFrame(weight, calories, date)
	*/

	entries := []Entry{
		{UserWeight: 180.3, Calories: 2400, Date: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.1, Calories: 2400, Date: time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.1, Calories: 2400, Date: time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.2, Calories: 2400, Date: time.Date(2023, 1, 8, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.15, Calories: 2400, Date: time.Date(2023, 1, 9, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.22, Calories: 2400, Date: time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.42, Calories: 2400, Date: time.Date(2023, 1, 11, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.4, Calories: 2400, Date: time.Date(2023, 1, 12, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.1, Calories: 2400, Date: time.Date(2023, 1, 13, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.2, Calories: 2400, Date: time.Date(2023, 1, 14, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.2, Calories: 2400, Date: time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.4, Calories: 2400, Date: time.Date(2023, 1, 16, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.39, Calories: 2400, Date: time.Date(2023, 1, 17, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.4, Calories: 2400, Date: time.Date(2023, 1, 18, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.3, Calories: 2400, Date: time.Date(2023, 1, 19, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.2, Calories: 2400, Date: time.Date(2023, 1, 20, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.2, Calories: 2400, Date: time.Date(2023, 1, 21, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.2, Calories: 2400, Date: time.Date(2023, 1, 22, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.1, Calories: 2400, Date: time.Date(2023, 1, 23, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.3, Calories: 2400, Date: time.Date(2023, 1, 24, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.1, Calories: 2400, Date: time.Date(2023, 1, 25, 0, 0, 0, 0, time.UTC)},
	}

	u.Phase.WeeklyChange = 0
	u.Phase.StartDate = time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)
	u.Phase.LastCheckedWeek = u.Phase.StartDate
	u.Phase.EndDate = time.Date(2023, time.January, 25, 0, 0, 0, 0, time.UTC)
	u.Phase.GoalCalories = 2400
	u.Phase.Name = "maintenance"

	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Start a new transaction.
	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
		return
	}

	status, total, err := checkMaintenance(tx, &u, &entries)

	fmt.Println(status)
	fmt.Println(total)
	fmt.Println(err)

	// Output:
	// 0
	// 0
	// <nil>
}

func ExampleCheckMaintenance_gained() {
	u := UserInfo{}
	u.Phase.StartDate = time.Date(2023, time.January, 06, 0, 0, 0, 0, time.UTC)

	/*
		weight := dataframe.NewSeriesString("weight", nil,
			"180.3", "180.1", "180.1", "180.2", "180.55", "180.82", "180.82", // Gained
			"181.4", "181.1", "182.2", "182.2", "181.4", "182.39", "182.4", // Gained
			"182.3", "183.2", "182.2", "183.2", "183.1", "183.3", "183.1")

		calories := dataframe.NewSeriesString("calories", nil,
			"2400", "2400", "2400", "2400", "2400", "2400", "2400",
			"2400", "2400", "2400", "2400", "2400", "2400", "2400",
			"2400", "2400", "2400", "2400", "2400", "2400", "2400")

		date := dataframe.NewSeriesString("date", nil,
			"2023-01-05", "2023-01-06", "2023-01-07", "2023-01-08", "2023-01-09", "2023-01-10", "2023-01-11",
			"2023-01-12", "2023-01-13", "2023-01-14", "2023-01-15", "2023-01-16", "2023-01-17", "2023-01-18",
			"2023-01-19", "2023-01-20", "2023-01-21", "2023-01-22", "2023-01-23", "2023-01-24", "2023-01-25")

		logs := dataframe.NewDataFrame(weight, calories, date)
	*/

	entries := []Entry{
		{UserWeight: 180.3, Calories: 2400, Date: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.1, Calories: 2400, Date: time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.1, Calories: 2400, Date: time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.2, Calories: 2400, Date: time.Date(2023, 1, 8, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.55, Calories: 2400, Date: time.Date(2023, 1, 9, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.82, Calories: 2400, Date: time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.82, Calories: 2400, Date: time.Date(2023, 1, 11, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.4, Calories: 2400, Date: time.Date(2023, 1, 12, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.1, Calories: 2400, Date: time.Date(2023, 1, 13, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 182.2, Calories: 2400, Date: time.Date(2023, 1, 14, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 182.2, Calories: 2400, Date: time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.4, Calories: 2400, Date: time.Date(2023, 1, 16, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 182.39, Calories: 2400, Date: time.Date(2023, 1, 17, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 182.4, Calories: 2400, Date: time.Date(2023, 1, 18, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 182.3, Calories: 2400, Date: time.Date(2023, 1, 19, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 183.2, Calories: 2400, Date: time.Date(2023, 1, 20, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 182.2, Calories: 2400, Date: time.Date(2023, 1, 21, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 183.2, Calories: 2400, Date: time.Date(2023, 1, 22, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 183.1, Calories: 2400, Date: time.Date(2023, 1, 23, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 183.3, Calories: 2400, Date: time.Date(2023, 1, 24, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 183.1, Calories: 2400, Date: time.Date(2023, 1, 25, 0, 0, 0, 0, time.UTC)},
	}

	u.Phase.WeeklyChange = 0
	u.Phase.StartDate = time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)
	u.Phase.LastCheckedWeek = u.Phase.StartDate
	u.Phase.EndDate = time.Date(2023, time.January, 25, 0, 0, 0, 0, time.UTC)
	u.Phase.GoalCalories = 2400
	u.Phase.Name = "maintain"
	u.Phase.Status = "active"

	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Start a new transaction.
	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
		return
	}

	err = setupTestConfigTables(tx)
	if err != nil {
		return
	}

	status, total, err := checkMaintenance(tx, &u, &entries)

	fmt.Println(status)
	fmt.Printf("%.2f\n", total)
	fmt.Println(err)

	// Output:
	// 1
	// 2.10
	// <nil>
}

func ExampleCheckMaintenance_lost() {
	u := UserInfo{}
	u.Phase.StartDate = time.Date(2023, time.January, 06, 0, 0, 0, 0, time.UTC)

	/*
		weight := dataframe.NewSeriesString("weight", nil,
			"182.4", "181.1", "181.2", "181.2", "181.4", "181.09", "181.0", // Lost
			"180.9", "180.7", "180.5", "180.5", "180.55", "180.22", "180.12", // Lost
			"182.3", "183.2", "182.2", "183.2", "183.1", "183.3", "183.1")

		calories := dataframe.NewSeriesString("calories", nil,
			"2400", "2400", "2400", "2400", "2400", "2400", "2400",
			"2400", "2400", "2400", "2400", "2400", "2400", "2400",
			"2400", "2400", "2400", "2400", "2400", "2400", "2400")

		date := dataframe.NewSeriesString("date", nil,
			"2023-01-05", "2023-01-06", "2023-01-07", "2023-01-08", "2023-01-09", "2023-01-10", "2023-01-11",
			"2023-01-12", "2023-01-13", "2023-01-14", "2023-01-15", "2023-01-16", "2023-01-17", "2023-01-18",
			"2023-01-19", "2023-01-20", "2023-01-21", "2023-01-22", "2023-01-23", "2023-01-24", "2023-01-25")

		logs := dataframe.NewDataFrame(weight, calories, date)
	*/

	entries := []Entry{
		{UserWeight: 182.4, Calories: 2400, Date: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.1, Calories: 2400, Date: time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.2, Calories: 2400, Date: time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.2, Calories: 2400, Date: time.Date(2023, 1, 8, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.4, Calories: 2400, Date: time.Date(2023, 1, 9, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.09, Calories: 2400, Date: time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.0, Calories: 2400, Date: time.Date(2023, 1, 11, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.9, Calories: 2400, Date: time.Date(2023, 1, 12, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.7, Calories: 2400, Date: time.Date(2023, 1, 13, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.5, Calories: 2400, Date: time.Date(2023, 1, 14, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.5, Calories: 2400, Date: time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.55, Calories: 2400, Date: time.Date(2023, 1, 16, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.22, Calories: 2400, Date: time.Date(2023, 1, 17, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.12, Calories: 2400, Date: time.Date(2023, 1, 18, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 182.3, Calories: 2400, Date: time.Date(2023, 1, 19, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 183.2, Calories: 2400, Date: time.Date(2023, 1, 20, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 182.2, Calories: 2400, Date: time.Date(2023, 1, 21, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 183.2, Calories: 2400, Date: time.Date(2023, 1, 22, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 183.1, Calories: 2400, Date: time.Date(2023, 1, 23, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 183.3, Calories: 2400, Date: time.Date(2023, 1, 24, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 183.1, Calories: 2400, Date: time.Date(2023, 1, 25, 0, 0, 0, 0, time.UTC)},
	}

	u.Phase.WeeklyChange = 0
	u.Phase.StartDate = time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)
	u.Phase.LastCheckedWeek = u.Phase.StartDate
	u.Phase.EndDate = time.Date(2023, time.January, 25, 0, 0, 0, 0, time.UTC)
	u.Phase.GoalCalories = 2400
	u.Phase.Name = "maintain"
	u.Phase.Status = "active"

	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Start a new transaction.
	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
		return
	}

	err = setupTestConfigTables(tx)
	if err != nil {
		return
	}

	status, total, err := checkMaintenance(tx, &u, &entries)

	fmt.Println(status)
	fmt.Printf("%.2f\n", total)
	fmt.Println(err)

	// Output:
	// -1
	// -2.28
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

	/*
		weight := dataframe.NewSeriesString("weight", nil,
			"180", "180.1", "180.2", "180.3", "180.3", "180.4", "180.5",
			"180.6", "180.5", "180.6", "180.7", "180.8", "180.0", "181",
			"181.1", "181.2", "181.3", "181.4", "181.5", "181.5", "181.5")

		calories := dataframe.NewSeriesString("calories", nil,
			"2400", "2400", "2400", "2400", "2400", "2400", "2400",
			"2400", "2400", "2400", "2400", "2400", "2500", "2400",
			"2500", "2500", "2500", "2550", "2550", "2450", "2500")

		date := dataframe.NewSeriesString("date", nil,
			"2023-01-05", "2023-01-06", "2023-01-07", "2023-01-08", "2023-01-09", "2023-01-10", "2023-01-11",
			"2023-01-12", "2023-01-13", "2023-01-14", "2023-01-15", "2023-01-16", "2023-01-17", "2023-01-18",
			"2023-01-19", "2023-01-20", "2023-01-21", "2023-01-22", "2023-01-23", "2023-01-24", "2023-01-25")

		logs := dataframe.NewDataFrame(weight, calories, date)
	*/

	entries := []Entry{
		{UserWeight: 180.0, Calories: 2400, Date: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.1, Calories: 2400, Date: time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.2, Calories: 2400, Date: time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.3, Calories: 2400, Date: time.Date(2023, 1, 8, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.3, Calories: 2400, Date: time.Date(2023, 1, 9, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.4, Calories: 2400, Date: time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.5, Calories: 2400, Date: time.Date(2023, 1, 11, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.6, Calories: 2400, Date: time.Date(2023, 1, 12, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.5, Calories: 2400, Date: time.Date(2023, 1, 13, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.6, Calories: 2400, Date: time.Date(2023, 1, 14, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.7, Calories: 2400, Date: time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.8, Calories: 2400, Date: time.Date(2023, 1, 16, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.0, Calories: 2500, Date: time.Date(2023, 1, 17, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.0, Calories: 2400, Date: time.Date(2023, 1, 18, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.1, Calories: 2500, Date: time.Date(2023, 1, 19, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.2, Calories: 2500, Date: time.Date(2023, 1, 20, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.3, Calories: 2500, Date: time.Date(2023, 1, 21, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.4, Calories: 2550, Date: time.Date(2023, 1, 22, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.5, Calories: 2550, Date: time.Date(2023, 1, 23, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.5, Calories: 2450, Date: time.Date(2023, 1, 24, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.5, Calories: 2500, Date: time.Date(2023, 1, 25, 0, 0, 0, 0, time.UTC)},
	}

	u.Phase.StartDate = time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)
	u.Phase.LastCheckedWeek = u.Phase.StartDate
	u.Phase.EndDate = time.Date(2023, time.January, 25, 0, 0, 0, 0, time.UTC)
	u.Phase.WeeklyChange = 0.5
	u.Phase.GoalCalories = 2400
	u.Phase.Name = "bulk"
	u.Phase.Status = "active"

	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Start a new transaction.
	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
		return
	}

	err = setupTestConfigTables(tx)
	if err != nil {
		return
	}

	status, avgTotal, err := checkBulkGain(tx, &u, &entries)

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

	/*
		weight := dataframe.NewSeriesString("weight", nil,
			"180", "180", "180", "180", "180", "180", "180", // Gained too little.
			"175", "175", "175", "175", "175", "175", "175", // Gained too little.
			"170", "170", "170", "170", "170", "170", "170")

		calories := dataframe.NewSeriesString("calories", nil,
			"2400", "2400", "2400", "2400", "2400", "2400", "2400",
			"2300", "2400", "2400", "2450", "2400", "2400", "2350",
			"2450", "2400", "2450", "2400", "2400", "2400", "2500")

		date := dataframe.NewSeriesString("date", nil,
			"2023-01-05", "2023-01-06", "2023-01-07", "2023-01-08", "2023-01-09", "2023-01-10", "2023-01-11",
			"2023-01-12", "2023-01-13", "2023-01-14", "2023-01-15", "2023-01-16", "2023-01-17", "2023-01-18",
			"2023-01-19", "2023-01-20", "2023-01-21", "2023-01-22", "2023-01-23", "2023-01-24", "2023-01-25")

		logs := dataframe.NewDataFrame(weight, calories, date)
	*/

	entries := []Entry{
		{UserWeight: 180.0, Calories: 2400, Date: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.0, Calories: 2400, Date: time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.0, Calories: 2400, Date: time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.0, Calories: 2400, Date: time.Date(2023, 1, 8, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.0, Calories: 2400, Date: time.Date(2023, 1, 9, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.0, Calories: 2400, Date: time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.0, Calories: 2400, Date: time.Date(2023, 1, 11, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 175.0, Calories: 2300, Date: time.Date(2023, 1, 12, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 175.0, Calories: 2400, Date: time.Date(2023, 1, 13, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 175.0, Calories: 2400, Date: time.Date(2023, 1, 14, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 175.0, Calories: 2450, Date: time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 175.0, Calories: 2400, Date: time.Date(2023, 1, 16, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 175.0, Calories: 2400, Date: time.Date(2023, 1, 17, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 175.0, Calories: 2350, Date: time.Date(2023, 1, 18, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.0, Calories: 2450, Date: time.Date(2023, 1, 19, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.0, Calories: 2400, Date: time.Date(2023, 1, 20, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.0, Calories: 2450, Date: time.Date(2023, 1, 21, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.0, Calories: 2400, Date: time.Date(2023, 1, 22, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.0, Calories: 2400, Date: time.Date(2023, 1, 23, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.0, Calories: 2400, Date: time.Date(2023, 1, 24, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.0, Calories: 2500, Date: time.Date(2023, 1, 25, 0, 0, 0, 0, time.UTC)},
	}

	u.Phase.StartDate = time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)
	u.Phase.LastCheckedWeek = u.Phase.StartDate
	u.Phase.EndDate = time.Date(2023, time.January, 25, 0, 0, 0, 0, time.UTC)
	u.Phase.WeeklyChange = 0.5
	u.Phase.GoalCalories = 2400
	u.Phase.Name = "bulk"
	u.Phase.Status = "active"

	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Start a new transaction.
	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
		return
	}

	err = setupTestConfigTables(tx)
	if err != nil {
		return
	}

	status, avgTotal, err := checkBulkGain(tx, &u, &entries)

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

	/*
		weight := dataframe.NewSeriesString("weight", nil,
			"170", "170", "170", "170", "170", "170", "170", // Gained too little.
			"170.2", "170.4", "170.4", "170.4", "170.6", "170.8", "170.8", // Gained too much.
			"171.0", "171.2", "171.2", "171.4", "171.4", "171.6", "171.8") // Gained too much.

		calories := dataframe.NewSeriesString("calories", nil,
			"2400", "2400", "2400", "2400", "2400", "2400", "2400",
			"2400", "2400", "2400", "2400", "2400", "2400", "2400",
			"2500", "2500", "2500", "2500", "2500", "2200", "2500")

		date := dataframe.NewSeriesString("date", nil,
			"2023-01-05", "2023-01-06", "2023-01-07", "2023-01-08", "2023-01-09", "2023-01-10", "2023-01-11",
			"2023-01-12", "2023-01-13", "2023-01-14", "2023-01-15", "2023-01-16", "2023-01-17", "2023-01-18",
			"2023-01-19", "2023-01-20", "2023-01-21", "2023-01-22", "2023-01-23", "2023-01-24", "2023-01-25")

		logs := dataframe.NewDataFrame(weight, calories, date)
	*/

	entries := []Entry{
		{UserWeight: 170.0, Calories: 2400, Date: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.0, Calories: 2400, Date: time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.0, Calories: 2400, Date: time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.0, Calories: 2400, Date: time.Date(2023, 1, 8, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.0, Calories: 2400, Date: time.Date(2023, 1, 9, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.0, Calories: 2400, Date: time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.0, Calories: 2400, Date: time.Date(2023, 1, 11, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.2, Calories: 2400, Date: time.Date(2023, 1, 12, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.4, Calories: 2400, Date: time.Date(2023, 1, 13, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.4, Calories: 2400, Date: time.Date(2023, 1, 14, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.4, Calories: 2400, Date: time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.6, Calories: 2400, Date: time.Date(2023, 1, 16, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.8, Calories: 2400, Date: time.Date(2023, 1, 17, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 170.8, Calories: 2400, Date: time.Date(2023, 1, 18, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 171.0, Calories: 2500, Date: time.Date(2023, 1, 19, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 171.2, Calories: 2500, Date: time.Date(2023, 1, 20, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 171.2, Calories: 2500, Date: time.Date(2023, 1, 21, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 171.4, Calories: 2500, Date: time.Date(2023, 1, 22, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 171.4, Calories: 2500, Date: time.Date(2023, 1, 23, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 171.6, Calories: 2200, Date: time.Date(2023, 1, 24, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 171.8, Calories: 2500, Date: time.Date(2023, 1, 25, 0, 0, 0, 0, time.UTC)},
	}

	u.Phase.StartDate = time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)
	u.Phase.LastCheckedWeek = u.Phase.StartDate
	u.Phase.EndDate = time.Date(2023, time.January, 25, 0, 0, 0, 0, time.UTC)
	u.Phase.WeeklyChange = 0.5
	u.Phase.GoalCalories = 2400
	u.Phase.Name = "bulk"
	u.Phase.Status = "active"

	// Connect to the test database
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Start a new transaction.
	tx, err := db.Beginx()
	if err != nil {
		log.Println(err)
		return
	}

	err = setupTestConfigTables(tx)
	if err != nil {
		return
	}

	status, total, err := checkBulkGain(tx, &u, &entries)

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
	u.Weight = 180 // lbs
	u.Height = 65  //
	u.Age = 30
	u.ActivityLevel = "light"
	bmr := Mifflin(&u)
	u.TDEE = TDEE(bmr, u.ActivityLevel)

	u.Phase.StartDate = time.Date(2023, time.January, 06, 0, 0, 0, 0, time.UTC)
	u.Phase.Duration = 8
	u.Phase.EndDate = calculateEndDate(u.Phase.StartDate, u.Phase.Duration)
	u.Phase.WeeklyChange = 0.75 // Desired weekly change in weight in pounds.
	u.Phase.GoalCalories = u.TDEE + (u.Phase.WeeklyChange * 500)
	u.Phase.LastCheckedWeek = u.Phase.StartDate
	setMinMaxMacros(&u)
	u.Macros.Protein, u.Macros.Carbs, u.Macros.Fats = calculateMacros(&u)

	avgWeekWeightChange := 0.50 // User is not gaining enough weight.

	addCals(&u, avgWeekWeightChange)

	// Output:
	// Adding to caloric surplus by 125.00 calories.
	// New calorie goal: 2842.09.
}

func ExampleTotalWeightChangeWeek() {
	u := UserInfo{}
	u.Phase.StartDate = time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)

	/*
		weight := dataframe.NewSeriesString("weight", nil,
			"180", "181", "183", "184", "185", "186")

		calories := dataframe.NewSeriesString("calories", nil,
			"2400", "2400", "2400", "2400", "2400", "2400")

		date := dataframe.NewSeriesString("date", nil,
			"2023-01-05", "2023-01-06", "2023-01-07", "2023-01-08", "2023-01-09", "2023-01-10")

		logs := dataframe.NewDataFrame(weight, calories, date)
	*/

	entries := []Entry{
		{UserWeight: 180.0, Calories: 2400, Date: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.0, Calories: 2400, Date: time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 183.0, Calories: 2400, Date: time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 184.0, Calories: 2400, Date: time.Date(2023, 1, 8, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 185.0, Calories: 2400, Date: time.Date(2023, 1, 9, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 186.0, Calories: 2400, Date: time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)},
	}

	start := time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)
	end := time.Date(2023, time.January, 11, 0, 0, 0, 0, time.UTC)

	avg, _, _ := totalWeightChangeWeek(&entries, start, end, &u)
	fmt.Println(avg)

	// Output:
	// 6
}

func ExampleFindEntryIdx() {
	/*
		weight := dataframe.NewSeriesString("weight", nil, "180", "182", "180.5", "181.1",
			"182.2", "182.1", "183.4", "183", "183.3", "183.2")
		calories := dataframe.NewSeriesString("calories", nil, "2410", "2490", "2573", "2400",
			"2408", "2499", "2550", "2570", "2600", "2599")
		date := dataframe.NewSeriesString("date", nil, "2023-01-05", "2023-01-06", "2023-01-07",
			"2023-01-08", "2023-01-09", "2023-01-10",
			"2023-01-11", "2023-01-12", "2023-01-13", "2023-01-14")
		logs := dataframe.NewDataFrame(weight, calories, date)
	*/

	entries := []Entry{
		{UserWeight: 180.0, Calories: 2410, Date: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 182.0, Calories: 2490, Date: time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.5, Calories: 2573, Date: time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 181.1, Calories: 2400, Date: time.Date(2023, 1, 8, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 182.2, Calories: 2408, Date: time.Date(2023, 1, 9, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 182.1, Calories: 2499, Date: time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 183.4, Calories: 2550, Date: time.Date(2023, 1, 11, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 183.0, Calories: 2570, Date: time.Date(2023, 1, 12, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 183.3, Calories: 2600, Date: time.Date(2023, 1, 13, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 183.2, Calories: 2599, Date: time.Date(2023, 1, 14, 0, 0, 0, 0, time.UTC)},
	}

	day := time.Date(2023, time.January, 8, 0, 0, 0, 0, time.UTC)

	i, err := findEntryIdx(&entries, day)

	fmt.Println(i)
	fmt.Println(err)

	// Output:
	// 3
	// <nil>
}

func ExampleGetPrecedingWeightToDay() {
	u := UserInfo{}
	u.Phase.StartDate = time.Date(2023, time.January, 06, 0, 0, 0, 0, time.UTC)

	/*
		weight := dataframe.NewSeriesString("weight", nil, "180", "182", "180.5")
		calories := dataframe.NewSeriesString("calories", nil, "2410", "2490", "2573")
		date := dataframe.NewSeriesString("date", nil, "2023-01-05", "2023-01-06", "2023-01-07")
		logs := dataframe.NewDataFrame(weight, calories, date)
	*/

	entries := []Entry{
		{UserWeight: 180.0, Calories: 2410, Date: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 182.0, Calories: 2490, Date: time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.5, Calories: 2573, Date: time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC)},
	}

	startIdx := 2 // Index of the succeeding date.

	w, err := getPrecedingWeightToDay(&u, &entries, 180.5, startIdx)

	fmt.Println(w)
	fmt.Println(err)

	// Output:
	// 182
	// <nil>
}

func ExampleGetPrecedingWeightToDay_beforePhase() {
	u := UserInfo{}
	u.Phase.StartDate = time.Date(2023, time.January, 06, 0, 0, 0, 0, time.UTC)

	/*
		weight := dataframe.NewSeriesString("weight", nil, "180", "182", "180.5")
		calories := dataframe.NewSeriesString("calories", nil, "2410", "2490", "2573")
		date := dataframe.NewSeriesString("date", nil, "2023-01-05", "2023-01-06", "2023-01-07")
		logs := dataframe.NewDataFrame(weight, calories, date)
	*/

	entries := []Entry{
		{UserWeight: 180.0, Calories: 2410, Date: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 182.0, Calories: 2490, Date: time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC)},
		{UserWeight: 180.5, Calories: 2573, Date: time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC)},
	}

	startIdx := 1 // Index of the succeeding date.

	w, err := getPrecedingWeightToDay(&u, &entries, 182, startIdx)

	fmt.Println(w)
	fmt.Println(err)

	// Output:
	// 180
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

func ExampleCalculateGoalWeight() {
	fmt.Println(calculateGoalWeight(180, 8, defaultBulkWeeklyChangePct))
	// Output:
	// 183.63
}

func ExampleSetRecommendedValues() {
	u := UserInfo{}
	u.Phase.StartDate = time.Date(2023, time.January, 01, 0, 0, 0, 0, time.UTC)
	setRecommendedValues(&u, 1.25, 8, 170, 2300)
	fmt.Println(u.Phase.WeeklyChange)
	fmt.Println(u.Phase.Duration)
	fmt.Println(u.Phase.GoalWeight)
	fmt.Println(u.Phase.GoalCalories)
	fmt.Println(u.Phase.LastCheckedWeek)

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

func ExampleValidateDateIsNotPast() {
	today := time.Now()
	date := today.AddDate(0, 0, 1)
	fmt.Println(validateDateIsNotPast(date))

	// Output:
	// true
}

func ExampleValidateDate() {
	dateStr := "2023-01-23"
	date, err := validateDateStr(dateStr)
	fmt.Println(date)
	fmt.Println(err)

	// Output:
	// 2023-01-23 00:00:00 +0000 UTC
	// <nil>
}

func TestValidateDate_parseError(t *testing.T) {
	dateStr := "2023 01 23"
	_, err := validateDateStr(dateStr)

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

func ExampleSummary() {
	u := UserInfo{}
	u.Weight = 180
	u.Height = 180
	u.Age = 30
	u.ActivityLevel = "light"
	bmr := Mifflin(&u)
	u.TDEE = TDEE(bmr, u.ActivityLevel)
	today := time.Now()

	/*
		var weightSeriesElements []interface{}
		var caloriesSeriesElements []interface{}
		var dateSeriesElements []interface{}

		weightVal := 184.0
		caloriesVal := 2300.0

		for i := 0; i < 28; i++ {
			dateVal := today.AddDate(0, 0, -27+i).Format(dateFormat)
			dateSeriesElements = append(dateSeriesElements, dateVal)

			weightVal -= 0.10
			weightSeriesElements = append(weightSeriesElements, strconv.FormatFloat(weightVal, 'f', 1, 64))

			caloriesVal -= 10
			caloriesSeriesElements = append(caloriesSeriesElements, strconv.Itoa(int(caloriesVal)))
		}

		weightSeries := dataframe.NewSeriesString("weight", nil, weightSeriesElements...)
		caloriesSeries := dataframe.NewSeriesString("calories", nil, caloriesSeriesElements...)
		dateSeries := dataframe.NewSeriesString("date", nil, dateSeriesElements...)

		logs := dataframe.NewDataFrame(weightSeries, caloriesSeries, dateSeries)
	*/

	entries := make([]Entry, 28)

	weightVal := 184.0
	caloriesVal := 2300.0

	for i := 0; i < 28; i++ {
		dateVal := today.AddDate(0, 0, -27+i)

		weightVal -= 0.10
		caloriesVal -= 10

		entries[i] = Entry{
			UserWeight: weightVal,
			Calories:   float64(caloriesVal),
			Date:       dateVal,
		}
	}

	/*
		weight := dataframe.NewSeriesString("weight", nil,
			"183.0", "183.2", "182.2", "182.3", "182.3", "182.4", "182.5",
			"181.6", "181.5", "181.6", "181.7", "180.8", "180.0", "181",
			"181.1", "181.2", "181.3", "181.4", "181.5", "181.5", "181.5",
			"181.1", "181.2", "181.3", "181.4", "181.5", "181.5", "181.5")

		calories := dataframe.NewSeriesString("calories", nil,
			"2012", "2090", "2150", "2200", "2100", "2100", "2200",
			"2200", "2100", "2100", "2200", "2300", "2300", "2300",
			"2200", "2350", "2100", "2100", "2250", "2100", "2150",
			"2200", "2350", "2100", "2100", "2250", "2100", "2150")

		date := dataframe.NewSeriesString("date", nil,
			"2023-06-02", "2023-06-03", "2023-06-04", "2023-06-05", "2023-06-06",
			"2023-06-07", "2023-06-08", "2023-06-09", "2023-06-10", "2023-06-11", "2023-06-12", "2023-06-13",
			"2023-06-14", "2023-06-15", "2023-06-16", "2023-06-17", "2023-06-18", "2023-06-19", "2023-06-20",
			"2023-06-21", "2023-06-22", "2023-06-23", "2023-06-24", "2023-06-25", "2023-06-26", "2023-06-27", "2023-06-28", "2023-06-29")

		logs := dataframe.NewDataFrame(weight, calories, date)
		logs.Append(nil, "181.5", "2200", today.Format(dateFormat))
	*/

	u.Weight = 181.20000000000016
	//u.Phase.StartDate, _ = time.Parse(dateFormat, logs.Series[dateCol].Value(0).(string))
	u.Phase.StartDate = entries[0].Date
	u.Phase.EndDate = today.AddDate(0, 0, 3)
	u.Phase.Status = "active"
	u.Phase.GoalCalories = 2200
	u.Phase.Name = "cut"
	u.Phase.StartWeight = 183.2
	u.Phase.GoalWeight = 178

	Summary(&u, &entries)

	// Output:
	// 0
}

func setupTestConfigTables(tx *sqlx.Tx) error {
	_, err := tx.Exec(`
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
		return err
	}
	return nil
}
