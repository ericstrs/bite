package calories

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rocketlaunchr/dataframe-go"
	"github.com/rocketlaunchr/dataframe-go/imports"
)

// Entry fields will be constructed from daily_weights and daily_foods
// table during runtime.
/*
type Entry struct {
	UserWeight float64 // User weight for a single day.
	UserCals   float64 // Consumed calories for a single day.
	Date       time.Time
	Protein    float64 // Consumed protein for a single day.
	Carbs      float64 // Consumed carbohydrate  for a single day.
	Fat        float64 // Consumed fat for a single day.
}
*/

// Nutrient are for portion size (100 serving unit)
type Entry struct {
	UserWeight float64   `db:"user_weight"`
	UserCals   float64   `db:"user_cals"`
	Date       time.Time `db:"date"`
	Protein    float64   `db:"protein"`
	Carbs      float64   `db:"carbs"`
	Fat        float64   `db:"fat"`
}

// GetAllEntries returns all the user's entries from the database.
func GetAllEntries(db *sqlx.DB) (*[]Entry, error) {
	// TODO: should probably get number_of_serving from `food_prefs` table
	//in its own SQL statement before hand.

	query := `
	SELECT
		-- Select date and user's weight for each day.
		dw.date,
  	dw.weight as user_weight,

		-- Calculate sum of calories and macros for each day.
		-- If nutrient is not logged for a particular day, its amount is treated as 0.
		-- Nutrient amount is multiplied by number of servings which has a default of 1.
  	SUM(CASE WHEN fn.nutrient_id = 1008 THEN fn.amount * df.number_of_servings ELSE 0 END) as user_cals,
  	SUM(CASE WHEN fn.nutrient_id = 1003 THEN fn.amount * df.number_of_servings ELSE 0 END) as protein,
  	SUM(CASE WHEN fn.nutrient_id = 1005 THEN fn.amount * df.number_of_servings ELSE 0 END) as carbs,
  	SUM(CASE WHEN fn.nutrient_id = 1004 THEN fn.amount * df.number_of_servings ELSE 0 END) as fat

		FROM daily_weights dw -- User's weight data.

			-- Join with daily food data on date.
  		-- Only join if food_id is not null (i.e., at least one food is logged for that day).
			JOIN daily_foods df ON dw.date = df.date AND df.food_id IS NOT NULL

			-- Join with food nutrients data on food_id.
			JOIN food_nutrients fn ON df.food_id = fn.food_id

  -- Only include specific nutrient_ids in the result.
	WHERE fn.nutrient_id IN (1008, 1003, 1005, 1004)

	GROUP BY 
	  -- Group by date and user weight to aggregate nutrition data by day.
		dw.date,
		dw.weight

	-- Only include groups where at least one food_id is not null
  -- (i.e., at least one food was logged for that day).
	HAVING SUM(df.food_id) IS NOT NULL

	-- Order results by date.
	ORDER BY dw.date`

	var entries []Entry
	err := db.Select(&entries, query)
	if err != nil {
		log.Fatal(err)
	}

	return &entries, nil
}

// ReadEntries reads user entries from CSV file into a dataframe.
func ReadEntries() (*dataframe.DataFrame, error) {
	// Does entries file exist?
	if _, err := os.Stat(EntriesFilePath); os.IsNotExist(err) {
		log.Println("ERROR: Entries file not found.")
		return nil, err
	}

	// Open entries file
	csvfile, err := os.Open(EntriesFilePath)
	if err != nil {
		log.Printf("ERROR: Couldn't open %s\n", EntriesFilePath)
		return nil, err
	}
	defer csvfile.Close()

	// Read entries from CSV into a dataframe.
	ctx := context.TODO()
	logs, err := imports.LoadFromCSV(ctx, csvfile)
	if err != nil {
		log.Printf("ERROR: Couldn't read %s\n", EntriesFilePath)
		return nil, err
	}

	return logs, nil
}

// checkInput checks if the user input is positive
func checkInput(n float64) error {
	if n < 0 {
		return errors.New("invalid number")
	}
	return nil
}

// promptCals prompts the user to enter caloric intake for the previous
// day.
func promptCals() (calories float64, err error) {
	fmt.Print("Enter caloric intake for the day: ")
	fmt.Scanln(&calories)

	return calories, checkInput(calories)
}

// Log appends a new entry to the csv file passed in as an agurment.
func Log(u *UserInfo, s string) error {
	var date time.Time
	var err error

	// Get user weight.
	u.Weight, err = getWeight(u.System)
	if err != nil {
		return err
	}

	// Get user calories for the day.
	cals, err := promptCals()
	if err != nil {
		return err
	}

	for {
		// Prompt user entry date.
		r := promptDate("Enter entry date (YYYY-MM-DD) [Press Enter for today's date]: ")

		// If user entered default date,
		if r == "" {
			// set date to today's date.
			date = time.Now()
			break
		}

		// Check if date is a valid date.
		date, err = validateDate(r)
		if err != nil {
			fmt.Printf("%v. Please try again.\n", err)
			continue
		}

		// Validate the date is not in the past.
		if date.Before(time.Now()) {
			fmt.Println("The entered date is in the past. Please enter today's date or a future date.")
			continue
		}

		break
	}

	// Save updated user info.
	err = saveUserInfo(u)
	if err != nil {
		return err
	}

	// Open file for append.
	f, err := os.OpenFile(s, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	if err != nil {
		log.Println(err)
		return err
	}

	// Append user calorie input to csv file.
	line := fmt.Sprintf("%.2f,%.2f,%s\n", u.Weight, cals, date.Format(dateFormat))
	_, err = f.WriteString(line)
	if err != nil {
		log.Println(err)
		return err
	}

	fmt.Println("Added entry.")

	return nil
}

// subset returns a subset of the dataframe containing the entries that
// were logged during an active diet phase.
func Subset(logs *dataframe.DataFrame, indices []int) *dataframe.DataFrame {
	s1 := dataframe.NewSeriesString("weight", nil)
	s2 := dataframe.NewSeriesString("calories", nil)
	s3 := dataframe.NewSeriesString("date", nil)
	s := dataframe.NewDataFrame(s1, s2, s3)

	for _, idx := range indices {
		row := logs.Row(idx, false, dataframe.SeriesIdx|dataframe.SeriesName)
		s.Append(nil, row)
	}

	return s
}

// getValidLogIndices creates and returns and int array containing the
// indices of the the valid log entries.
//
// Assumptions:
// * Diet phase activity has been checked. That is, this function should
// not be called for a diet phase that is not currently active.
func GetValidLogIndices(u *UserInfo, logs *dataframe.DataFrame) []int {
	today := time.Now()

	var validIndices []int
	for i := 0; i < logs.NRows(); i++ {
		date, err := time.Parse(dateFormat, logs.Series[dateCol].Value(i).(string))
		if err != nil {
			log.Println("ERROR: Couldn't parse date:", err)
			return nil
		}

		// Only consider dates that fall somewhere inbetween the diet
		// start date and the current date.
		if (date.After(u.Phase.StartDate) || isSameDay(date, u.Phase.StartDate)) && (date.Before(today) || isSameDay(date, today)) {
			validIndices = append(validIndices, i)
		}
	}

	return validIndices
}
