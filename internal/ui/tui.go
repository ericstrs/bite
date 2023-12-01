package ui

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jmoiron/sqlx"
	"github.com/oneseIf/bite"
	"github.com/rivo/tview"
)

const (
	dateFormat = "2006-01-02"
	resultsFmt = "%-5.1f %-2s x %-2.1f serving |%-3.0f cals|protein: %.1fg, carbs: %.1fg, fat: %.1fg\n"
)

type SearchUI struct {
	// app is a reference to the tview application
	app *tview.Application

	// pages supports pop up forms.
	pages *tview.Pages

	// inputField is a UI element for text input, allowing users to enter
	// their search queries. The entered text is used for search operations.
	inputField *tview.InputField

	// list represents a table view in the UI, used to display search
	// results. Each row in the table can correspond to a different zettel
	// title, tag line, or zettel.
	list *tview.Table

	// db is the database connection.
	db *sqlx.DB

	// screenWidth holds the width of the screen in characters.
	screenWidth int

	// Item being searched for.
	item string
}

// NewSearchUI creates and initializes a new SearchUI.
func NewSearchUI(db *sqlx.DB, query, item string) *SearchUI {
	sui := &SearchUI{
		app:         tview.NewApplication(),
		inputField:  tview.NewInputField(),
		list:        tview.NewTable(),
		db:          db,
		item:        item,
		screenWidth: 50,
	}

	sui.setupUI(query)

	return sui
}

// setupUI configures the UI elements.
func (sui *SearchUI) setupUI(query string) {
	sui.globalInput()

	// Update screen width before drawing. This won't affect the current
	// drawing, it sets the width for the next draw operation.
	sui.app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		sui.screenWidth, _ = screen.Size()
		return false
	})

	switch sui.item {
	case `food`:
		sui.setupFoodUI(query)
	case `meal`:
		sui.setupMealUI(query)
	default:
		log.Printf("Item %q not supported\n", sui.item)
		return
	}

	sui.inputField.SetLabel("Search: ").
		SetFieldWidth(30)
	sui.inputField.SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	sui.list.SetBorder(true)
	style := tcell.StyleDefault.Background(tcell.Color107).Foreground(tcell.ColorBlack)
	sui.list.SetSelectedStyle(style)
	sui.listInput()
	sui.inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			sui.list.SetSelectable(true, false)
			sui.app.SetFocus(sui.list)
		}
	})

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(sui.inputField, 1, 0, true).
		AddItem(sui.list, 0, 1, false)

	sui.pages = tview.NewPages().
		AddPage("", flex, true, true)

	sui.app.SetRoot(sui.pages, true)
}

func (sui *SearchUI) setupFoodUI(query string) {
	t := "Loading recent entries in the background. Begin typing to search, or wait to browse."
	foods := []bite.Food{bite.Food{Name: t, FoodMacros: &bite.FoodMacros{}}}
	go func() {
		var err error
		foods, err = bite.GetRecentlyLoggedFoods(sui.db, bite.SearchLimit)
		if err != nil {
			log.Printf("couldn't get recently logged foods: %v\n", err)
			return
		}
		sui.app.QueueUpdateDraw(func() {
			text := sui.inputField.GetText()
			if text == "" {
				sui.updateFoodsList(foods)
			}
		})
	}()

	sui.ipInputFood(&foods)

	switch query {
	case "":
		sui.updateFoodsList(foods)
	default:
		sui.inputField.SetText(query)
	}
}

func (sui *SearchUI) setupMealUI(query string) {
	t := "Loading recent meals in the background. Begin typing to search, or wait to browse."
	meals := []bite.Meal{bite.Meal{Name: t}}
	go func() {
		var err error
		meals, err = bite.GetMealsWithRecentFirst(sui.db)
		if err != nil {
			log.Printf("couldn't get recently logged meals: %v\n", err)
			return
		}
		sui.app.QueueUpdateDraw(func() {
			text := sui.inputField.GetText()
			if text == "" {
				sui.displayMeals(meals)
			}
		})
	}()

	sui.ipInputMeal(&meals)

	switch query {
	case "":
		sui.displayMeals(meals)
	default:
		sui.inputField.SetText(query)
	}
}

// globalInput handles input capture for the application.
func (sui *SearchUI) globalInput() {
	sui.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			sui.app.Stop()
		}
		return event
	})
}

// ipInputFood handles input capture for the inputField.
//
// It interprets the following key bindings and triggers corresponding
// actions:
//
//   - Enter: Sets focus to results list.
//   - Ctrl+Enter: Uses current search query as title for new zettel.
//   - Esc: Exits the search interface.
func (sui *SearchUI) ipInputFood(foods *[]bite.Food) {
	var debounceTimer *time.Timer
	sui.inputField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		/*
			// If ctrl+enter pressed,
			if event.Modifiers() == 2 && event.Rune() == 10 {
				text := sui.inputField.GetText()
				_ = text
				sui.app.Stop()
				// TODO: Create new food with current input field text.
			}
		*/
		return event
	})
	sui.inputField.SetChangedFunc(func(text string) {
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceTimer = time.AfterFunc(100*time.Millisecond, func() {
			go func() {
				latestText := sui.inputField.GetText()
				if latestText == "" {
					sui.app.QueueUpdateDraw(func() {
						sui.updateFoodsList(*foods)
					})
					return
				}
				results := sui.performFoodSearch(latestText)
				sui.app.QueueUpdateDraw(func() {
					sui.updateFoodsList(results)
				})
			}()
		})
	})
}

// ipInputMeal handles input capture for the inputField.
//
// It interprets the following key bindings and triggers corresponding
// actions:
//
//   - Enter: Sets focus to results list.
//   - Ctrl+Enter: Uses current search query as title for new zettel.
//   - Esc: Exits the search interface.
func (sui *SearchUI) ipInputMeal(meals *[]bite.Meal) {
	var debounceTimer *time.Timer
	sui.inputField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		/*
			// If ctrl+enter pressed,
			if event.Modifiers() == 2 && event.Rune() == 10 {
				text := sui.inputField.GetText()
				_ = text
				sui.app.Stop()
				// TODO: Create new meal with current input field text.
			}
		*/
		return event
	})
	sui.inputField.SetChangedFunc(func(text string) {
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceTimer = time.AfterFunc(100*time.Millisecond, func() {
			go func() {
				latestText := sui.inputField.GetText()
				if latestText == "" {
					sui.app.QueueUpdateDraw(func() {
						sui.displayMeals(*meals)
					})
					return
				}
				meals := sui.performMealSearch(latestText)
				sui.app.QueueUpdateDraw(func() {
					sui.updateMealsList(meals)
				})
			}()
		})
	})
}

func (sui *SearchUI) displayFoods(foods []bite.Food) {
	row := 0
	for i := 0; i < len(foods); i++ {
		f := foods[i]
		s := f.Name
		sui.list.SetCell(row, 0, tview.NewTableCell(s).
			SetReference(&f))
		row++
	}
	sui.list.ScrollToBeginning()
}

func (sui *SearchUI) displayMeals(meals []bite.Meal) {
	row := 0
	for i := 0; i < len(meals); i++ {
		m := meals[i]
		s := m.Name
		sui.list.SetCell(row, 0, tview.NewTableCell(s).
			SetReference(&m))
		row++
	}
	sui.list.ScrollToBeginning()
}

// performFoodSearch gets foods to update the foods list.
func (sui *SearchUI) performFoodSearch(query string) []bite.Food {
	if query == "" {
		return []bite.Food{}
	}

	var err error
	var foods []bite.Food
	recent := strings.HasPrefix(query, `recent:`)
	switch recent {
	case false:
		foods, err = bite.SearchFoods(sui.db, query)
	case true:
		var recent []bite.Food
		recent, err = bite.GetRecentlyLoggedFoods(sui.db, bite.SearchLimit)
		query = strings.TrimSpace(query[len("recent:"):])
		for _, f := range recent {
			// Case-insensitive search for food names
			if strings.Contains(strings.ToLower(f.Name), strings.ToLower(query)) {
				foods = append(foods, f)
			}
		}
	}

	if err != nil {
		foods = []bite.Food{bite.Food{Name: `Incorrect syntax`, FoodMacros: &bite.FoodMacros{}}}
	}
	return foods
}

// performMealSearch gets meals to update the meals list.
func (sui *SearchUI) performMealSearch(query string) []bite.Meal {
	if query == "" {
		return []bite.Meal{}
	}
	meals, err := bite.SearchMeals(sui.db, query)
	if err != nil {
		meals = []bite.Meal{bite.Meal{Name: `Incorrect syntax`}}
	}
	return meals
}

// updateFoodsList updates the results list with a given slice of zettels.
func (sui *SearchUI) updateFoodsList(foods []bite.Food) {
	list := sui.list
	list.Clear()
	if len(foods) == 0 {
		list.SetCellSimple(0, 0, "No matches found.")
		return
	}
	row := 0
	for i := 0; i < len(foods); i++ {
		f := foods[i]
		s := "[powderblue]" + f.Name + "[white]"
		list.SetCell(row, 0, tview.NewTableCell(s).
			SetReference(&f))
		row++
		line := fmt.Sprintf(resultsFmt, f.ServingSize, f.ServingUnit,
			f.NumberOfServings, f.Calories, f.FoodMacros.Protein,
			f.FoodMacros.Carbs, f.FoodMacros.Fat)
		list.SetCell(row, 0, tview.NewTableCell(line).
			SetSelectable(false))
		row++
		list.SetCell(row, 0, tview.NewTableCell("").
			SetSelectable(false))
		row++
	}
	sui.list.ScrollToBeginning()
}

// updateMealsList updates the results list with a given slice of meals.
func (sui *SearchUI) updateMealsList(meals []bite.Meal) {
	list := sui.list
	list.Clear()
	if len(meals) == 0 {
		list.SetCellSimple(0, 0, "No matches found.")
		return
	}
	row := 0
	for i := 0; i < len(meals); i++ {
		m := meals[i]
		s := m.Name
		list.SetCell(row, 0, tview.NewTableCell(s).
			SetReference(&m))
		row++
	}
	sui.list.ScrollToBeginning()
}

// listInput handles input capture for the list.
//
// It interprets the following key bindings and triggers corresponding
// actions:
//
//   - enter: Log selected item.
//   - e: edit selected item.
//   - d: delete selected item.
//   - ESC, q: Exits the search interface.
//   - H: Move to the top of the visible window.
//   - M: Move to the center of the visible window.
//   - L: Move to bottom of the visible window.
//   - space: Page down
//   - b: Page up
//
// If selection is on first result and 'k' is pressed, set focus on
// input field.
func (sui *SearchUI) listInput() {
	sui.list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			row, col := sui.list.GetSelection()
			cell := sui.list.GetCell(row, col)
			switch f := cell.GetReference().(type) {
			case *bite.Food:
				tx, err := sui.db.Beginx()
				defer tx.Rollback()
				if err != nil {
					log.Println("couldn't create transaction: ", err)
					return nil
				}
				date := time.Now()
				// Log selected food to the food log database table. Taking into
				// account food preferences.
				if err := bite.AddFoodEntry(tx, f, date); err != nil {
					log.Println("couldn't add food log: ", err)
					return nil
				}
				tx.Commit()
				return nil
			case *bite.Meal:
				// TODO: log selected meal
			default:
				log.Printf("Table cell doesn't reference bite.Food or bite.Meal: %T\n", f)
			}
			return nil
		case tcell.KeyEscape:
			sui.app.Stop()
		default:
			switch event.Rune() {
			case 'H': // move to top of the visible window
				row, _ := sui.list.GetOffset()
				sui.list.Select(row, 0)
				return nil
			case 'M': // move to middle of the visible window
				row, _ := sui.list.GetOffset()
				_, _, _, height := sui.list.GetInnerRect()
				sui.list.Select(row+height/2, 0)
				return nil
			case 'L': // move to bottom of the visible window
				row, _ := sui.list.GetOffset()
				_, _, _, height := sui.list.GetInnerRect()
				sui.list.Select(row+height-1, 0)
				return nil
			case 'b': // page up (Ctrl-B)
				return tcell.NewEventKey(tcell.KeyCtrlB, 0, tcell.ModNone)
			case ' ': // page down
				row, _ := sui.list.GetOffset()
				_, _, _, height := sui.list.GetInnerRect()
				newRow := row + height
				if newRow > sui.list.GetRowCount()-1 {
					newRow = sui.list.GetRowCount() - 1
				}
				sui.list.SetOffset(newRow, 0)
				sui.list.Select(newRow, 0)
				return nil
			case 'e': // Edit
				row, col := sui.list.GetSelection()
				cell := sui.list.GetCell(row, col)
				switch f := cell.GetReference().(type) {
				case *bite.Food:
					form := sui.editFoodForm(f)
					sui.showModal(form)
					return nil
				case *bite.Meal:
					// TODO: edit selected meal
					return nil
				default:
					log.Printf("Table cell doesn't reference bite.Food or bite.Meal: %T\n", f)
				}
			case 'd': // delete
				row, col := sui.list.GetSelection()
				cell := sui.list.GetCell(row, col)
				switch f := cell.GetReference().(type) {
				case *bite.Food:
					form := sui.confirmFoodDeletion(f)
					sui.showModal(form)
				case *bite.Meal:
				}
			case 'q': // quit app
				sui.app.Stop()
			case 'k':
				row, _ := sui.list.GetSelection()
				if row == 0 {
					sui.list.SetSelectable(false, false)
					sui.app.SetFocus(sui.inputField)
				}
			}
		}
		return event
	})
}

// editFoodForm creates and returns a tview form for editing a food.
func (sui *SearchUI) editFoodForm(f *bite.Food) *tview.Form {
	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle("Edit Food")

	name := f.Name
	brandName := f.BrandName
	price := f.Price
	protein := f.FoodMacros.Protein
	carbs := f.FoodMacros.Carbs
	fat := f.FoodMacros.Fat
	hhServing := f.HouseholdServing

	servingSize := f.ServingSize
	numServings := f.NumberOfServings

	// Define the input fields for the forms and update field variables if
	// user makes any changes to the default values.
	form.AddInputField("Name", name, 20, nil, func(text string) {
		f.Name = text
	})
	form.AddInputField("Brand Name", brandName, 20, nil, func(text string) {
		f.BrandName = text
	})
	form.AddInputField("Serving Size", fmt.Sprintf("%.1f", servingSize), 20, nil, func(text string) {
		num, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return
		}
		f.ServingSize = num
	})
	form.AddInputField("Num Servings", fmt.Sprintf("%.1f", numServings), 20, nil, func(text string) {
		num, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return
		}
		f.NumberOfServings = num
	})

	form.AddInputField("Protein", fmt.Sprintf("%.1f", protein), 20, nil, func(text string) {
		num, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return
		}
		f.FoodMacros.Protein = num
	})
	form.AddInputField("Carbs", fmt.Sprintf("%.1f", carbs), 20, nil, func(text string) {
		num, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return
		}
		f.FoodMacros.Carbs = num
	})
	form.AddInputField("Fat", fmt.Sprintf("%.1f", fat), 20, nil, func(text string) {
		num, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return
		}
		f.FoodMacros.Fat = num
	})
	if hhServing != "" {
		form.AddInputField("Household Serving", hhServing, 20, nil, func(text string) {
			f.HouseholdServing = text
		})
	}
	form.AddInputField("Price", fmt.Sprintf("%.1f", price), 20, nil, func(text string) {
		num, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return
		}
		f.Price = num
	})

	form.AddButton("Save", func() {
		tx, err := sui.db.Beginx()
		defer tx.Rollback()
		if err != nil {
			log.Println("couldn't create transaction: ", err)
			return
		}

		if err := updateFoodTable(tx, *f); err != nil {
			log.Println("couldn't update food table: ", err)
			return
		}
		fp := &bite.FoodPref{}
		fp.FoodID = f.ID
		fp.ServingSize = f.ServingSize
		fp.NumberOfServings = f.NumberOfServings

		// Update food prefs table
		if err := bite.UpdateFoodPrefs(tx, fp); err != nil {
			log.Println("couldn't update food preferences: ", err)
			return
		}

		f.Calories = bite.CalculateCalories(f.FoodMacros.Protein, f.FoodMacros.Carbs, f.FoodMacros.Fat)
		if err := bite.UpdateFoodNutrients(sui.db, tx, f); err != nil {
			log.Println("couldn't update food nutrients: ", err)
			return
		}
		tx.Commit()

		sui.updateSelectedFood(*f)

		sui.closeModal()
	})

	form.AddButton("Cancel", func() {
		sui.closeModal()
	})

	return form
}

func (sui *SearchUI) confirmFoodDeletion(f *bite.Food) *tview.Form {
	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle("Confirm Food Deletion")

	form.AddButton("Confirm", func() {
		tx, err := sui.db.Beginx()
		defer tx.Rollback()
		if err != nil {
			log.Println("couldn't create transaction: ", err)
			return
		}
		if err := bite.DeleteFood(tx, f.ID); err != nil {
			log.Println("couldn't delete food: ", err)
			return
		}
		tx.Commit()
		sui.closeModal()
	})

	form.AddButton("Cancel", func() {
		sui.closeModal()
	})

	return form
}

// updateFoodTable partially updates one food from the foods table
func updateFoodTable(tx *sqlx.Tx, food bite.Food) error {
	const query = `
  UPDATE foods SET
  food_name = $1, serving_size = $2, serving_unit = $3
  WHERE food_id = $4
  `
	_, err := tx.Exec(query, food.Name, food.ServingSize, food.ServingUnit,
		food.ID)
	if err != nil {
		return fmt.Errorf("Failed to update food: %v", err)
	}

	return nil
}

// updateSelectedFood updates the selected food in the results list.
func (sui *SearchUI) updateSelectedFood(f bite.Food) {
	row, col := sui.list.GetSelection()
	cell := sui.list.GetCell(row, col)
	s := "[powderblue]" + f.Name + "[white]"
	cell.SetText(s)
	line := fmt.Sprintf(resultsFmt, f.ServingSize, f.ServingUnit,
		f.NumberOfServings, f.Calories, f.FoodMacros.Protein,
		f.FoodMacros.Carbs, f.FoodMacros.Fat)
	descCell := sui.list.GetCell(row+1, col)
	descCell.SetText(line)

}

// closeModal removes the modal page
func (sui *SearchUI) closeModal() {
	sui.pages.RemovePage("modal")
	sui.app.SetFocus(sui.list)
}

// showModal sets up a modal grid for the given form and displays it.
func (sui *SearchUI) showModal(form *tview.Form) {
	// Returns a new primitive which puts the provided primitive in the center and
	// sets its size to the given width and height.
	modal := func(p tview.Primitive, width, height int) tview.Primitive {
		return tview.NewGrid().
			SetColumns(0, width, 0).
			SetRows(0, height, 0).
			AddItem(p, 1, 1, 1, 1, 0, 0, true)
	}

	m := modal(form, 40, 30)
	sui.pages.AddPage("modal", m, true, true)
	sui.app.SetFocus(m)
}

// Run starts the TUI application.
func (sui *SearchUI) Run() error {
	return sui.app.Run()
}
