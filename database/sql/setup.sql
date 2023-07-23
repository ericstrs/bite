-- foods contains static information about foods.
CREATE TABLE IF NOT EXISTS foods (
  food_id INTEGER PRIMARY KEY,
  food_name TEXT NOT NULL,
  serving_size REAL NOT NULL,
  serving_unit TEXT NOT NULL,
  household_serving TEXT NOT NULL
);

-- meals contains static information about the meals. A meal is a
-- collection of foods.
CREATE TABLE IF NOT EXISTS meals (
    meal_id INTEGER PRIMARY KEY,
    meal_name TEXT NOT NULL
);

-- user_foods contains the user's food consumption
-- logs.
CREATE TABLE IF NOT EXISTS daily_foods (
  id INTEGER PRIMARY KEY,
  food_id INTEGER REFERENCES foods(food_id) NOT NULL,
  meal_ID INTEGER REFERENCES meals(meal_id),
  date DATE NOT NULL,
  number_of_servings REAL DEFAULT 1 NOT NULL
);

-- user_meals contains the user's meal consumption logs.
CREATE TABLE IF NOT EXISTS daily_meals (
  id INTEGER PRIMARY KEY,
  meal_id INTEGER REFERENCES meals(meal_id),
  date DATE NOT NULL
);

-- daily_weights contains the users daily weight and date of the entry.
CREATE TABLE IF NOT EXISTS daily_weights (
  id INTEGER PRIMARY KEY,
  date DATE NOT NULL,
  weight REAL NOT NULL
);

-- meal_foods relates meals to the foods the contain.
CREATE TABLE IF NOT EXISTS meal_foods (
  meal_id INTEGER REFERENCES meals(meal_id),
  food_id INTEGER REFERENCES foods(food_id),
  number_of_servings REAL DEFAULT 1 NOT NULL
);

-- nutrients stores the nurtients that a food can be comprised of.
CREATE TABLE IF NOT EXISTS nutrients (
  nutrient_id INTEGER PRIMARY KEY,
  nutrient_name TEXT NOT NULL,
  unit_name TEXT NOT NULL
);

-- food_nutrient_derivation stores the procedure indicating how a food
-- nutrient value was obtained.
CREATE TABLE IF NOT EXISTS food_nutrient_derivation (
  id INT PRIMARY KEY,
  code VARCHAR(255) NOT NULL,
  description TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS food_nutrients (
  id INTEGER PRIMARY KEY,
  food_id INTEGER NOT NULL,
  nutrient_id INTEGER NOT NULL,
  amount REAL NOT NULL,
  derivation_id REAL NOT NULL,
  FOREIGN KEY (food_id) REFERENCES foods(food_id),
  FOREIGN KEY (nutrient_id) REFERENCES nutrients(nutrients_id),
  FOREIGN KEY (derivation_id) REFERENCES food_nutrient_derivation(id)
);

CREATE TABLE IF NOT EXISTS food_prefs (
  food_id INTEGER PRIMARY KEY,
  serving_size REAL,
  number_of_servings REAL DEFAULT 1 NOT NULL,
  FOREIGN KEY(food_id) REFERENCES foods(food_id)
);

CREATE TABLE IF NOT EXISTS meal_food_prefs (
  meal_id INTEGER,
  food_id INTEGER,
  serving_size REAL,
  number_of_servings REAL DEFAULT 1 NOT NULL,
  PRIMARY KEY(meal_id, food_id),
  FOREIGN KEY(food_id) REFERENCES foods(food_id),
  FOREIGN KEY(meal_id) REFERENCES meals(meal_id)
);
