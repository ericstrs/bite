# Bite

Food and weight tracker.

This program contains the following features:

* Daily calorie and weight tracking.
* Cut, maintenance, and bulk phase tracker.

## Finding your activity level

|Activity Level|Description|
|--------------|-----------|
|Sedentary|No exercise and stationary lifestyle|
|Lightly active|Exercise 1-2 days per week|
|Moderately active|Exercise 3-5 days per week|
|Active|Exercise 6-7 days per week|
|Very active|Exercise 2x per day|

## Command line arguments

cmd
 |
 |- log
 |   |- weight
 |   |- food
 |   |- meal
 |   |- update
 |   |   |- weight
 |   |   |- food
 |   |- delete
 |   |   |- weight
 |   |   |- food
 |   |- show
 |   |   |- all
 |   |   |- weight
 |   |   |- food
 |
 |- add
 |   |- food
 |   |- meal
 |
 |- delete
 |    |- food
 |    |- meal
 |
 |- update
 |    |- user
 |    |- food
 |    |- meal
 |    |  |- add
 |    |  |- delete
 |
 |- summary
 |     |- phase
 |     |- diet
 |     |  |- all
 |     |  |- day
 |     |- user
 |
 |- Stop/Start
 |      |- phase

## TODO

[] Let user modify current diet:
  [] Add user capability to stop a diet phase and begin a new one. They shouldn't have to be prompted for user details.
  [] Add user ability to change macro ratio
[X] Handle trends that counter the aim of the user (gaining weight/ losing weight)
[] Handle negative weight trends:
  [X] Work out how to deal with inconsistent user entries.
  [X] Work out how you measure whats a week/month.
[] Summary:
  [] Streak feature.
  [] Add Monthly view of adherence.
  [] Add macros pie chart.
  [] Are you on track towards reaching your desired weight goal?
[] Record history
  [] Add log of completed diet phases with related data.

## Sources

* U.S. Department of Agriculture, Agricultural Research Service. FoodData Central, 2019. fdc.nal.usda.gov.
