-- Create temporary table for food data
CREATE TEMPORARY TABLE temp_foods (
  fdc_id TEXT,
  data_type TEXT,
  description TEXT,
  food_category_id TEXT,
  publication_date TEXT
);

.mode csv
.import food.csv temp_foods

-- Create temporary table for branded food data
CREATE TEMPORARY TABLE temp_branded_foods (
  fdc_id TEXT,
  brand_owner TEXT,
  brand_name TEXT,
  subbrand_name TEXT,
  gtin_upc TEXT,
  ingredients TEXT,
  not_a_significant_source_of TEXT,
  serving_size REAL,
  serving_size_unit TEXT,
  household_serving_fulltext TEXT, -- This is the serving size in food amount. its emtpy for most.
  branded_food_category TEXT,
  data_source TEXT,
  package_weight TEXT,
  modified_date TEXT,
  available_date TEXT,
  market_country TEXT,
  discontinued_date TEXT,
  preparation_state_code TEXT,
  trade_channel TEXT,
  short_description TEXT
);

.import branded_food.csv temp_branded_foods

-- Create temporary table for food attribute data
-- Only used for second option in COALESCE statement for the food
-- table insert.
CREATE TEMPORARY TABLE temp_food_attributes (
  id TEXT,
  fdc_id TEXT,
  seq_num TEXT,
  food_attribute_type_id TEXT,
  name TEXT,
  value TEXT
);

.import food_attribute.csv temp_food_attributes

-- Insert foods into the food table from both temporary tables
INSERT INTO foods(food_id, food_name, serving_size, serving_unit, household_serving)
SELECT
  CAST(food.fdc_id AS INTEGER),
  food.description,
  CAST(branded.serving_size AS REAL),
  COALESCE(branded.serving_size_unit, 'g'), -- assume grams if no unit given
  COALESCE(branded.household_serving_fulltext, '') -- empty string if no household serving is given
FROM temp_foods AS food
INNER JOIN temp_branded_foods AS branded ON food.fdc_id = branded.fdc_id;

DROP TABLE temp_foods;
DROP TABLE temp_branded_foods;
DROP TABLE temp_food_attributes;

CREATE TEMPORARY TABLE temp_nutrients(
    "id" TEXT,
    "name" TEXT,
    "unit_name" TEXT,
    "nutrient_nbr" TEXT,
    "rank" TEXT
);

.import nutrient.csv temp_nutrients

INSERT INTO nutrients(nutrient_id, nutrient_name, unit_name)
SELECT CAST(id AS INTEGER), name, unit_name FROM temp_nutrients;

DROP TABLE temp_nutrients;

CREATE TEMPORARY TABLE temp_food_nutrients (
  "id" TEXT,
  "fdc_id" TEXT,
  "nutrient_id" TEXT,
  "amount" TEXT,
  "data_points" TEXT,
  "derivation_id" TEXT,
  "min" TEXT,
  "max" TEXT,
  "median" TEXT,
  "loq" TEXT,
  "footnote" TEXT,
  "min_year_acquired" TEXT
);

.import food_nutrient.csv temp_food_nutrients

-- Import derivation data into existing table
.import food_nutrient_derivation.csv food_nutrient_derivation

INSERT INTO food_nutrients(id, food_id, nutrient_id, amount, derivation_id)
SELECT
  CAST(tfn.id AS INTEGER),
  CAST(tfn.fdc_id AS INTEGER),
  CAST(tfn.nutrient_id AS INTEGER),
  CAST(tfn.amount AS INTEGER),
  CAST(tfn.derivation_id AS INT)
FROM
  temp_food_nutrients tfn
JOIN
  foods f ON tfn.fdc_id = f.food_id;

DROP TABLE temp_food_nutrients;

--------------------[ Data Modifications ]------------------------------

-- Ensure every food has a calorie entry for the food_nutrients table.
INSERT INTO food_nutrients(food_id, nutrient_id, amount, derivation_id)
SELECT
    f.food_id,
    1008, -- 1008 is the nutrient_id for calories
    (4 * COALESCE(fnp.amount, 0)) + (4 * COALESCE(fnc.amount, 0)) + (9 * COALESCE(fnf.amount, 0)), -- Missing macros default to 0
    49 -- 49 is the derivation id for "calculated" nutrients
FROM
    foods f
    LEFT JOIN food_nutrients fnp ON f.food_id = fnp.food_id AND fnp.nutrient_id = 1003 -- 1003 is nutrient_id for protein
    LEFT JOIN food_nutrients fnc ON f.food_id = fnc.food_id AND fnc.nutrient_id = 1005 -- 1005 is nutrient_id for carbohydrates
    LEFT JOIN food_nutrients fnf ON f.food_id = fnf.food_id AND fnf.nutrient_id = 1004 -- 1004 is nutrient_id for fat
WHERE
    f.food_id NOT IN (SELECT food_id FROM food_nutrients WHERE nutrient_id = 1008);