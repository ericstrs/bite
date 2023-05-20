# Calories

Food and weight tracker.

This program contains several parts:

* Food and weight tracking
* Daily and weekly feedback.
  * Are you on track towards reaching your desired weight goal?
Bulk, cut, and patience phase tracker.
  * Pie chart

Activity level:

|Sedentary|\*1.2|
|Lightly active|\*1.375|
|Moderately active|\*1.55|
|Active|\*1.725|
|Very active|\*1.9|

Sedentary|\*1.2|
Lightly active|\*1.375|
Moderately active|\*1.55|
Active|\*1.725|
Very active|\*1.9|

## TODO

* Add user info file
  * How do you let the user know they don't have a info file?
  * Think default is prompt user for the user info, but if they pass file then skip the prompt.
* Handle trends that counter the aim of the user (gaining weight/ losing weight)
* Implement macro nutrients
  * Using `menu()` to select macro nutrient split. Then create a function to handle a CLI argument for when the user wants to know the split.
* Streak feature.
* Monthly view of adherence.
* Add summary feature
  * Pie chart
