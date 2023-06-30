package calories

import (
	"fmt"

	"github.com/rocketlaunchr/dataframe-go"
)

func ExampleSubset() {
	s1 := dataframe.NewSeriesString("weight", nil, "170", "170", "170", "170", "170", "170", "170", "170")
	s2 := dataframe.NewSeriesString("calories", nil, "2400", "2400", "2400", "2400", "2400", "2400", "2400", "2400")
	s3 := dataframe.NewSeriesString("date", nil, "2023-01-01", "2023-01-02", "2023-01-03", "2023-01-04", "2023-01-05", "2023-01-06", "2023-01-07", "2023-01-08")
	df := dataframe.NewDataFrame(s1, s2, s3)

	indices := []int{0, 2, 4} // indices we're interested in

	s := Subset(df, indices)
	fmt.Println(s)

	// Output:
	// +-----+--------+----------+------------+
	// |     | WEIGHT | CALORIES |    DATE    |
	// +-----+--------+----------+------------+
	// | 0:  |  170   |   2400   | 2023-01-01 |
	// | 1:  |  170   |   2400   | 2023-01-03 |
	// | 2:  |  170   |   2400   | 2023-01-05 |
	// +-----+--------+----------+------------+
	// | 3X3 | STRING |  STRING  |   STRING   |
	// +-----+--------+----------+------------+
}

/*
func ExampleGetValidLogIndices() {
  u := UserInfo{}

  var weightSeriesElements []interface{}
  var caloriesSeriesElements []interface{}
  var dateSeriesElements []interface{}

  weightVal := 184.0
  caloriesVal := 2300.0

  today := time.Now()

	// Initialize dataframe with 18 days prior to today.
  for i := 0; i < 18; i++ {
    dateVal := today.AddDate(0, 0, -17+i).Format(dateFormat)
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

	// Set starting date a few indices past 0; this simulates entries that
	// were logged before a diet phase began.
  u.Phase.StartDate, _ = time.Parse(dateFormat, logs.Series[dateCol].Value(10).(string))
	// Set end date to some arbitrary point past last logged entry (today)
  u.Phase.EndDate = today.AddDate(0, 0, 7)
  u.Phase.Active = true

  fmt.Println(getValidLogIndices(&u, logs))

  // Output:
  // [10 11 12 13 14 15 16]
}
*/
