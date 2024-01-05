# Bite

Bite is a tool to locally track your daily food intake and monitor diet phases.

## Features

* Smart diet phase tracking: Automatically adjust calorie and macro goals based on a given phase (weight loss, weight gain, maintenance).
* Food and user weight logging.
* Build custom meals.
* FTS food search.

## Finding your activity level

|Activity Level|Description|
|--------------|-----------|
|Sedentary|No exercise and stationary lifestyle|
|Lightly active|Exercise 1-2 days per week|
|Moderately active|Exercise 3-5 days per week|
|Active|Exercise 6-7 days per week|
|Very active|Exercise 2x per day|

## Install

Dependencies:

* tview library
* sqlite3
* USDA food database: [Full Download of All Data Types, April 2023 Release](https://fdc.nal.usda.gov/download-datasets.html#bkmk-1)
  * Run `setup.sql` and `import.sql` scripts to create sqlite database tables and import the USDA food data.

The command can be built from source or directly installed:

```
go install github.com/justericg/bite/cmd/bite@latest
```

## Embedded Documentation

Usage, controls, and other documentation has been embedded into the source code. See the source or run the application with the `help` command.

## Sources

* U.S. Department of Agriculture, Agricultural Research Service. FoodData Central, 2019. fdc.nal.usda.gov.
