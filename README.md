# Calories

Calorie/weight tracker.

This program contains several parts:

* Prompt user input for daily food and weight.
  * What is the users weight for day?
  * What was the total caloric intake for the previous day?
  * What was macro split the previous day?
  * Write to file.
    * `cal log edit`
    * `cal log print`
* Print out feedback for the user. What feedback does the user need to know?
  * Are they on track of the weight goal?
  * What is the desired macro split?
  * What is the current

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
* Implement macro nutrients
  * Using `menu()` to select macro nutrient split. Then create a function to handle a CLI argument for when the user wants to know the split.

