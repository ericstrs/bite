# Calories

Calorie/weight tracker using R.

* Uses data from a CSV file that includes information on weight.
* If `log` CLI argument,
  * prompt for yesterdays caloric intake
  * prompt for weight that was measured in the morning.
  * call `Sys.Date()`
  * Append this data to csv file

Activity level:

|Sedentary|\*1.2|
|Lightly active|\*1.375|
|Moderately active|\*1.55|
|Active|\*1.725|
|Very active|\*1.9|

## TODO

* Add a `generate()` function to create the `person.Rdata` file. This will allow for state to be updated.
  * If CLI argument is `set` then call `generate()`.
  * If `person.Rdata` does not exist, then let the user know and exit program.
* Handle trends that counter the aim of the user (gaining weight/ losing weight)
* Using `menu()` to select micronutrient split. Then create a function to handle a CLI argument for when the user wants to know the split.
* Plot TDEE overtime. This is an attempt to visualize data fit the true TDEE.

