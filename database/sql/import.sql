-- Create temporary table for food data
CREATE TEMPORARY TABLE temp_foods (
  fdc_id TEXT,
  data_type TEXT,
  description TEXT,
  food_category_id TEXT,
  publication_date TEXT
);

.mode csv
.import ./data/food.csv temp_foods

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

.import ./data/branded_food.csv temp_branded_foods

-- Insert foods into the food table from both temporary tables
INSERT INTO foods(food_id, food_name, serving_size, serving_unit, household_serving, brand_name)
SELECT
  CAST(food.fdc_id AS INTEGER),
  food.description,
  CAST(branded.serving_size AS REAL),
  COALESCE(branded.serving_size_unit, 'g'), -- assume grams if no unit given
  COALESCE(branded.household_serving_fulltext, ''), -- empty string if no household serving is given
  branded.brand_name
FROM temp_foods AS food
INNER JOIN temp_branded_foods AS branded ON food.fdc_id = branded.fdc_id;

DROP TABLE temp_branded_foods;

-- Create temporary table for food portion information for foundation
-- foods.
CREATE TEMPORARY TABLE temp_food_portion (
  id TEXT,
  fdc_id TEXT,
  seq_num TEXT,
  amount TEXT,
  measure_unit_id TEXT,
  portion_description TEXT,
  modifier TEXT,
  gram_weight TEXT,
  data_points TEXT,
  footnote TEXT,
  min_year_acquired TEXT
);

.import ./data/food_portion.csv temp_food_portion

-- Create temporary table for measure units to retreive fields for
-- foundation foods.
CREATE TEMPORARY TABLE temp_measure_unit (
  id TEXT,
  name TEXT
);

.import ./data/measure_unit.csv temp_measure_unit

-- Insert foundation foods into the foods table
INSERT INTO foods(food_id, food_name, serving_size, serving_unit, household_serving, brand_name)
SELECT
  CAST(food.fdc_id AS INTEGER),
  food.description,
  CAST(fp.gram_weight AS REAL),
  'g',
  COALESCE(fp.amount || ' ' || mu.name, ''),
  'foundation food'
FROM temp_foods AS food
JOIN temp_food_portion AS fp ON food.fdc_id = fp.fdc_id AND fp.seq_num = '1'
LEFT JOIN temp_measure_unit AS mu ON fp.measure_unit_id = mu.id
WHERE food.data_type = 'foundation_food';

-- Insert sr legacy foods into the foods table
INSERT INTO foods(food_id, food_name, serving_size, serving_unit, household_serving, brand_name)
SELECT
  CAST(food.fdc_id AS INTEGER),
  food.description,
  CAST(fp.gram_weight AS REAL),
  'g',
  COALESCE(fp.amount || ' ' || mu.name, ''),
  'reference'
FROM temp_foods AS food
JOIN temp_food_portion AS fp ON food.fdc_id = fp.fdc_id AND fp.seq_num = '1'
LEFT JOIN temp_measure_unit AS mu ON fp.measure_unit_id = mu.id
WHERE food.data_type = 'sr_legacy_food';

DROP TABLE temp_food_portion;
DROP TABLE temp_measure_unit;
DROP TABLE temp_foods;

CREATE TEMPORARY TABLE temp_nutrients(
    "id" TEXT,
    "name" TEXT,
    "unit_name" TEXT,
    "nutrient_nbr" TEXT,
    "rank" TEXT
);

.import ./data/nutrient.csv temp_nutrients

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

.import ./data/food_nutrient.csv temp_food_nutrients

-- Import derivation data into existing table
.import ./data/food_nutrient_derivation.csv food_nutrient_derivation

INSERT INTO food_nutrients(id, food_id, nutrient_id, amount, derivation_id)
SELECT
  CAST(tfn.id AS INTEGER),
  CAST(tfn.fdc_id AS INTEGER),
  CAST(tfn.nutrient_id AS INTEGER),
  CAST(tfn.amount AS INTEGER),
  CAST(tfn.derivation_id AS INT)
FROM
  temp_food_nutrients tfn
INNER JOIN
  foods f ON tfn.fdc_id = f.food_id;

DROP TABLE temp_food_nutrients;

--------------------[ Data Modifications ]------------------------------

-- Create a temporary table holding the IDs of foods that do not have
-- macros logged
CREATE TEMPORARY TABLE temp_food_ids AS
SELECT food_id
FROM foods
WHERE food_id NOT IN (
  SELECT DISTINCT food_id
  FROM food_nutrients
  WHERE nutrient_id IN (1003, 1004, 1005)
);

-- Delete foods that don't have any macros logged from foods table
DELETE FROM foods
WHERE food_id IN (
  SELECT food_id
  FROM temp_food_ids
);

-- Insert food columns that will be used to find food names
INSERT INTO foods_fts (food_id, food_name, brand_name)
SELECT
  food_id,
  food_name,
  brand_name
FROM foods;

-- Delete food nutrients that don't have any macros logged from food_nutrients table
DELETE FROM food_nutrients
WHERE food_id IN (
  SELECT food_id
  FROM temp_food_ids
);

DROP TABLE temp_food_ids;

-- Then, ensure every food has a calorie entry for the food_nutrients table.
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
    LEFT JOIN food_nutrients fn8 ON f.food_id = fn8.food_id AND fn8.nutrient_id = 1008 -- 1008 is nutrient_id for Energy (KCAL)
WHERE
    fn8.food_id IS NULL;


------------------------ [ Triggers ] ---------------------------------

CREATE TRIGGER insert_food_fts
  after INSERT on foods
BEGIN
  INSERT INTO foods_fts (food_id, food_name, brand_name)
  VALUES (NEW.food_id, NEW.food_name, NEW.brand_name);
END;

CREATE TRIGGER update_food_fts
  after UPDATE on foods
BEGIN
  UPDATE foods_fts
  SET 
    food_name = NEW.food_name,
    brand_name = NEW.brand_name
  WHERE food_id = NEW.food_id;
END;

CREATE TRIGGER delete_food_fts
  after DELETE on foods
BEGIN
  DELETE FROM foods_fts
  WHERE food_id = OLD.food_id;
END;
