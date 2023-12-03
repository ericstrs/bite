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
	dateFormat   = "2006-01-02"
	resultsFmt   = "%-5.1f %-2s x %-2.1f serving  |%-3.0f cals| protein: %.1fg, carbs: %.1fg, fat: %.1fg\n"
	mfResultsFmt = "  %-5.1f %-2s x %-2.1f serving %6.0f %10.1fg %13.1fg %11.1fg\n"
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

	// messages stores log messages that will get printed to stdout.
	messages []string

	selecting    bool
	selectedFood *bite.Food
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
		messages:    []string{},
	}

	sui.setupUI(query)

	return sui
}

// setupUI configures the search UI elements.
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

// setupSelectUI configures the search UI elements to support selecting
// a single food.
func (sui *SearchUI) setupSelectUI() *tview.Flex {
	sui.globalInput()

	sui.setupFoodUI("")

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

	return flex
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
			form := sui.errorForm("couldn't get recently logged meals", err)
			sui.showModal(form)
			return
		}
		sui.app.QueueUpdateDraw(func() {
			text := sui.inputField.GetText()
			if text == "" {
				sui.updateMealsList(meals)
			}
		})
	}()

	sui.ipInputMeal(&meals)

	switch query {
	case "":
		sui.updateMealsList(meals)
	default:
		sui.inputField.SetText(query)
	}
}

// globalInput handles input capture for the application.
//
// It interprets the following key bindings and triggers corresponding
// actions:
//
//   - Esc: Exits the search interface.
func (sui *SearchUI) globalInput() {
	sui.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			sui.app.Stop()
			for _, message := range sui.messages {
				fmt.Println(message)
			}
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
func (sui *SearchUI) ipInputFood(foods *[]bite.Food) {
	var debounceTimer *time.Timer
	sui.inputField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// If ctrl+enter pressed,
		if event.Modifiers() == 2 && event.Rune() == 10 {
			text := sui.inputField.GetText()
			form := sui.addFoodForm(text)
			sui.showModal(form)
		}
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
		// If ctrl+enter pressed,
		if event.Modifiers() == 2 && event.Rune() == 10 {
			// Create new meal with current input field text.
			text := sui.inputField.GetText()

			tx, err := sui.db.Beginx()
			defer tx.Rollback()
			if err != nil {
				form := sui.errorForm("couldn't create transaction: ", err)
				sui.showModal(form)
				return nil
			}

			_, err = bite.InsertMeal(tx, text)
			if err != nil {
				form := sui.errorForm("couldn't create new meal: ", err)
				sui.showModal(form)
				return nil
			}

			tx.Commit()

			sui.messages = append(sui.messages, fmt.Sprintf("Created new meal %q.\n", text))

			var meals []bite.Meal
			switch text == "" {
			case true:
				meals, err = bite.GetMealsWithRecentFirst(sui.db)
				if err != nil {
					form := sui.errorForm("couldn't get recently logged meals", err)
					sui.showModal(form)
					return nil
				}
			case false:
				meals = sui.performMealSearch(text)
			}
			sui.updateMealsList(meals)
		}
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
						sui.updateMealsList(*meals)
					})
					return
				}
				results := sui.performMealSearch(latestText)
				sui.app.QueueUpdateDraw(func() {
					sui.updateMealsList(results)
				})
			}()
		})
	})
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
		var s string
		switch f.BrandName == "" {
		case true:
			s = fmt.Sprintf("[powderblue]%s[white]", f.Name)
		case false:
			s = fmt.Sprintf("[powderblue]%s (%s)[white]", f.Name, f.BrandName)
		}
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
		s := "[powderblue]" + m.Name + "[white]"
		list.SetCell(row, 0, tview.NewTableCell(s).
			SetReference(&m))
		row++
		for j, _ := range m.Foods {
			mf := m.Foods[j]
			list.SetCell(row, 0, tview.NewTableCell("* "+mf.Food.Name).
				SetSelectable(true).
				SetReference(&mf))
			row++
			line := fmt.Sprintf(mfResultsFmt, mf.ServingSize, mf.Food.ServingUnit,
				mf.NumberOfServings, mf.Food.Calories, mf.Food.FoodMacros.Protein,
				mf.Food.FoodMacros.Carbs, mf.Food.FoodMacros.Fat)
			list.SetCell(row, 0, tview.NewTableCell(line).
				SetSelectable(false))
			row++
		}
		line := fmt.Sprintf("TOTAL: %24.1f cals %5.1fg protein %5.1fg carbs %5.1fg fat",
			m.Cals, m.Protein, m.Carbs, m.Fats)
		list.SetCell(row, 0, tview.NewTableCell(line).
			SetSelectable(false))
		row++
		list.SetCell(row, 0, tview.NewTableCell("").
			SetSelectable(false))
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
		case tcell.KeyEnter: // Log item
			row, col := sui.list.GetSelection()
			cell := sui.list.GetCell(row, col)

			tx, err := sui.db.Beginx()
			defer tx.Rollback()
			if err != nil {
				form := sui.errorForm("couldn't create transaction: ", err)
				sui.showModal(form)
				return nil
			}
			date := time.Now()

			switch i := cell.GetReference().(type) {
			case *bite.Food:
				if sui.selecting {
					return event
				}
				// Log selected food to the food log database table. Taking into
				// account food preferences.
				if err := bite.AddFoodEntry(tx, i, date); err != nil {
					form := sui.errorForm("couldn't add food log", err)
					sui.showModal(form)
					return nil
				}
				tx.Commit()
				sui.messages = append(sui.messages, "Logged food \""+i.Name+"\"")
			case *bite.Meal:
				// Log selected meal to the meal log database table. Taking into
				// account food preferences.
				if err := bite.AddMealEntry(tx, i.ID, date); err != nil {
					form := sui.errorForm("", err)
					sui.showModal(form)
					return nil
				}

				// Bulk insert the foods that make up the meal into the daily_foods table.
				if err := bite.AddMealFoodEntries(tx, i.ID, i.Foods, date); err != nil {
					form := sui.errorForm("", err)
					sui.showModal(form)
					return nil
				}

				tx.Commit()
				for _, mf := range i.Foods {
					sui.messages = append(sui.messages, "Logged food \""+mf.Name+"\"")
				}
			case *bite.MealFood:
				// TODO: log selected food
			default:
				form := sui.errorForm(fmt.Sprintf("Table cell doesn't reference bite.Food or bite.Meal: %T\n", i), nil)
				sui.showModal(form)
			}
			return nil
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
			case 'e': // Edit
				row, col := sui.list.GetSelection()
				cell := sui.list.GetCell(row, col)
				switch i := cell.GetReference().(type) {
				case *bite.Food:
					form := sui.editFoodForm(i)
					sui.showModal(form)
				case *bite.Meal:
					form := sui.editMealForm(i)
					sui.showModal(form)
				case *bite.MealFood:
					form := sui.editMealFoodForm(i)
					sui.showModal(form)
				default:
					form := sui.errorForm(fmt.Sprintf("Table cell doesn't reference type: %T\n", i), nil)
					sui.showModal(form)
				}
			case 'a': // Add food to meal
				row, col := sui.list.GetSelection()
				cell := sui.list.GetCell(row, col)
				switch i := cell.GetReference().(type) {
				case *bite.Meal: // Search and select food to add.
					ssui := &SearchUI{
						app:          sui.app,
						pages:        sui.pages,
						inputField:   tview.NewInputField(),
						list:         tview.NewTable(),
						db:           sui.db,
						item:         `food`,
						screenWidth:  50,
						messages:     []string{},
						selecting:    true,
						selectedFood: &bite.Food{},
					}

					ssui.list.SetSelectedFunc(func(row, col int) {
						var selectedFood bite.Food
						cell := ssui.list.GetCell(row, col)
						switch f := cell.GetReference().(type) {
						case *bite.Food:
							selectedFood = *f
						default:
							form := sui.errorForm(fmt.Sprintf("Table cell doesn't reference type: %T\n", i), nil)
							sui.showModal(form)
							return
						}

						// Remove the "select" page and set focus back to the original list
						sui.pages.RemovePage("select")
						sui.app.SetFocus(sui.list)

						// Insert food into meal food table.
						tx, err := sui.db.Beginx()
						defer tx.Rollback()
						if err != nil {
							form := sui.errorForm("couldn't create transaction: ", err)
							sui.showModal(form)
							return
						}
						if err := bite.InsertMealFood(tx, i.ID, selectedFood.ID); err != nil {
							form := sui.errorForm("couldn't insert meal food: ", err)
							sui.showModal(form)
							return
						}
						tx.Commit()
						sui.messages = append(sui.messages, fmt.Sprintf("Added food %q to meal %q.", selectedFood.Name, i.Name))

						var meals []bite.Meal
						text := sui.inputField.GetText()
						switch text == "" {
						case true:
							meals, err = bite.GetMealsWithRecentFirst(sui.db)
							if err != nil {
								form := sui.errorForm("couldn't get recently logged meals", err)
								sui.showModal(form)
								return
							}
						case false:
							meals = sui.performMealSearch(text)
						}
						sui.updateMealsList(meals)
					})

					flex := ssui.setupSelectUI()
					sui.pages.AddAndSwitchToPage("select", flex, true)
				}
			case 'd': // delete
				row, col := sui.list.GetSelection()
				cell := sui.list.GetCell(row, col)
				switch i := cell.GetReference().(type) {
				case *bite.Food:
					form := sui.confirmFoodDeletion(i)
					sui.showModal(form)
				case *bite.Meal:
					form := sui.confirmMealDeletion(i)
					sui.showModal(form)
				case *bite.MealFood:
					form := sui.mealFoodDeleteForm(i)
					sui.showModal(form)
				}
			case 'q': // quit app
				sui.app.Stop()
				for _, message := range sui.messages {
					fmt.Println(message)
				}
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
		name = text
	})
	form.AddInputField("Brand Name", brandName, 20, nil, func(text string) {
		brandName = text
	})
	form.AddInputField("Serving Size", fmt.Sprintf("%.1f", servingSize), 20, nil, func(text string) {
		num, err := strconv.ParseFloat(text, 64)
		if err != nil {
			num = 0
		}
		servingSize = num
	})
	form.AddInputField("Num Servings", fmt.Sprintf("%.1f", numServings), 20, nil, func(text string) {
		num, err := strconv.ParseFloat(text, 64)
		if err != nil {
			num = 0
		}
		numServings = num
	})
	form.AddInputField("Protein", fmt.Sprintf("%.1f", protein), 20, nil, func(text string) {
		num, err := strconv.ParseFloat(text, 64)
		if err != nil {
			num = 0
		}
		protein = num
	})
	form.AddInputField("Carbs", fmt.Sprintf("%.1f", carbs), 20, nil, func(text string) {
		num, err := strconv.ParseFloat(text, 64)
		if err != nil {
			num = 0
		}
		carbs = num
	})
	form.AddInputField("Fat", fmt.Sprintf("%.1f", fat), 20, nil, func(text string) {
		num, err := strconv.ParseFloat(text, 64)
		if err != nil {
			num = 0
		}
		fat = num
	})
	if hhServing != "" {
		form.AddInputField("Household Serving", hhServing, 20, nil, func(text string) {
			hhServing = text
		})
	}
	form.AddInputField("Price", fmt.Sprintf("%.1f", price), 20, nil, func(text string) {
		num, err := strconv.ParseFloat(text, 64)
		if err != nil {
			num = 0
		}
		price = num
	})

	form.AddButton("Save", func() {
		f.Name = name
		f.BrandName = brandName
		f.Price = price
		f.FoodMacros.Protein = protein
		f.FoodMacros.Carbs = carbs
		f.FoodMacros.Fat = fat
		f.HouseholdServing = hhServing
		f.ServingSize = servingSize
		f.NumberOfServings = numServings

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

// editMealForm creates and returns a tview form for editing a meal.
func (sui *SearchUI) editMealForm(m *bite.Meal) *tview.Form {
	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle("Edit Meal")

	name := m.Name

	// Define the input fields for the forms and update field variables if
	// user makes any changes to the default values.
	form.AddInputField("Name", name, 20, nil, func(text string) {
		name = text
	})

	form.AddButton("Save", func() {
		m.Name = name

		tx, err := sui.db.Beginx()
		defer tx.Rollback()
		if err != nil {
			log.Println("couldn't create transaction: ", err)
			return
		}

		if err := bite.UpdateMeal(tx, *m); err != nil {
			log.Println("couldn't update meal: ", err)
			return
		}
		tx.Commit()

		sui.updateSelectedMeal(*m)
		sui.closeModal()
	})

	form.AddButton("Cancel", func() {
		sui.closeModal()
	})

	return form

}

// editMealFoodForm creates and returns a tview form for editing a food.
func (sui *SearchUI) editMealFoodForm(mf *bite.MealFood) *tview.Form {
	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle("Edit Meal Food")

	servingSize := mf.ServingSize
	numServings := mf.NumberOfServings

	// Define the input fields for the forms and update field variables if
	// user makes any changes to the default values.
	form.AddInputField("Serving Size", fmt.Sprintf("%.1f", servingSize), 20, nil, func(text string) {
		num, err := strconv.ParseFloat(text, 64)
		if err != nil {
			num = 0
		}
		servingSize = num
	})
	form.AddInputField("Num Servings", fmt.Sprintf("%.1f", numServings), 20, nil, func(text string) {
		num, err := strconv.ParseFloat(text, 64)
		if err != nil {
			num = 0
		}
		numServings = num
	})

	form.AddButton("Save", func() {
		mf.ServingSize = servingSize
		mf.NumberOfServings = numServings

		tx, err := sui.db.Beginx()
		defer tx.Rollback()
		if err != nil {
			log.Println("couldn't create transaction: ", err)
			return
		}

		mfp := bite.MealFoodPref{}
		mfp.FoodID = mf.Food.ID
		mfp.MealID = int64(mf.MealID)
		mfp.ServingSize = mf.ServingSize
		mfp.NumberOfServings = mf.NumberOfServings

		if err := bite.UpdateMealFoodPrefs(tx, mfp); err != nil {
			log.Println("couldn't update meal food preferences: ", err)
			return
		}

		tx.Commit()
		sui.updateSelectedMealFood(*mf)
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

		sui.messages = append(sui.messages, fmt.Sprintf("Deleted food %q", f.Name))

		var foods []bite.Food
		text := sui.inputField.GetText()
		switch text == "" {
		case true:
			foods, err = bite.GetRecentlyLoggedFoods(sui.db, bite.SearchLimit)
			if err != nil {
				log.Printf("couldn't get recently logged foods: %v\n", err)
				return
			}
		case false:
			foods = sui.performFoodSearch(text)
		}
		sui.updateFoodsList(foods)

		sui.closeModal()
	})

	form.AddButton("Cancel", func() {
		sui.closeModal()
	})

	return form
}

func (sui *SearchUI) confirmMealDeletion(m *bite.Meal) *tview.Form {
	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(fmt.Sprintf("Delete meal %q?", m.Name))

	form.AddButton("Confirm", func() {
		tx, err := sui.db.Beginx()
		defer tx.Rollback()
		if err != nil {
			log.Println("couldn't create transaction: ", err)
			return
		}
		// Remove meal from the database.
		if err := bite.DeleteMeal(tx, m.ID); err != nil {
			log.Println("couldn't delete meal: ", err)
			return
		}
		tx.Commit()

		sui.messages = append(sui.messages, fmt.Sprintf("Deleted meal %q", m.Name))

		var meals []bite.Meal
		text := sui.inputField.GetText()
		switch text == "" {
		case true:
			meals, err = bite.GetMealsWithRecentFirst(sui.db)
			if err != nil {
				form := sui.errorForm("couldn't get recently logged meals", err)
				sui.showModal(form)
				return
			}
		case false:
			meals = sui.performMealSearch(text)
		}
		sui.updateMealsList(meals)

		sui.closeModal()
	})

	form.AddButton("Cancel", func() {
		sui.closeModal()
	})
	return form
}

func (sui *SearchUI) mealFoodDeleteForm(mf *bite.MealFood) *tview.Form {
	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(fmt.Sprintf("Delete meal food %q?", mf.Food.Name))

	form.AddButton("Confirm", func() {
		tx, err := sui.db.Beginx()
		defer tx.Rollback()
		if err != nil {
			log.Println("couldn't create transaction: ", err)
			return
		}
		if err := bite.DeleteMealFood(tx, mf.MealID, mf.Food.ID); err != nil {
			log.Println("couldn't delete meal food: ", err)
			return
		}
		tx.Commit()

		sui.messages = append(sui.messages, fmt.Sprintf("Deleted meal food %q", mf.Name))

		var meals []bite.Meal
		text := sui.inputField.GetText()
		switch text == "" {
		case true:
			meals, err = bite.GetMealsWithRecentFirst(sui.db)
			if err != nil {
				form := sui.errorForm("couldn't get recently logged meals", err)
				sui.showModal(form)
				return
			}
		case false:
			meals = sui.performMealSearch(text)
		}
		sui.updateMealsList(meals)

		sui.closeModal()
	})

	form.AddButton("Cancel", func() {
		sui.closeModal()
	})
	return form
}

func (sui *SearchUI) addFoodForm(name string) *tview.Form {
	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle("Add New Food")

	showingErr := false
	f := bite.Food{FoodMacros: &bite.FoodMacros{}}

	brandName := ""
	price := 0.0
	protein := 0.0
	carbs := 0.0
	fat := 0.0
	hhServing := ""
	servingUnit := "g"
	servingSize := 0.0
	numServings := 1.0

	// Define the input fields for the forms and update field variables if
	// user makes any changes to the default values.
	form.AddInputField("Name", name, 20, nil, func(text string) {
		name = text
	})
	form.AddInputField("Brand Name", brandName, 20, nil, func(text string) {
		brandName = text
	})
	form.AddInputField("Serving Size", fmt.Sprintf("%.1f", servingSize), 20, nil, func(text string) {
		num, err := strconv.ParseFloat(text, 64)
		if err != nil {
			num = 0
		}
		servingSize = num
	})
	form.AddInputField("Serving Unit", servingUnit, 20, nil, func(text string) {
		servingUnit = text
	})
	form.AddInputField("Num Servings", fmt.Sprintf("%.1f", numServings), 20, nil, func(text string) {
		num, err := strconv.ParseFloat(text, 64)
		if err != nil {
			num = 0
		}
		numServings = num
	})

	form.AddInputField("Protein", fmt.Sprintf("%.1f", protein), 20, nil, func(text string) {
		num, err := strconv.ParseFloat(text, 64)
		if err != nil {
			num = 0
		}
		protein = num
	})
	form.AddInputField("Carbs", fmt.Sprintf("%.1f", carbs), 20, nil, func(text string) {
		num, err := strconv.ParseFloat(text, 64)
		if err != nil {
			num = 0
		}
		carbs = num
	})
	form.AddInputField("Fat", fmt.Sprintf("%.1f", fat), 20, nil, func(text string) {
		num, err := strconv.ParseFloat(text, 64)
		if err != nil {
			num = 0
		}
		fat = num
	})
	if hhServing != "" {
		form.AddInputField("Household Serving", hhServing, 20, nil, func(text string) {
			hhServing = text
		})
	}
	form.AddInputField("Price", fmt.Sprintf("%.1f", price), 20, nil, func(text string) {
		num, err := strconv.ParseFloat(text, 64)
		if err != nil {
			num = 0
		}
		price = num
	})

	form.AddButton("Save", func() {
		if !showingErr {
			if name == "" || servingSize == 0 || servingUnit == "" || numServings == 0 {
				errorMsg := "Please enter non-zero values for fields: Name, Serving Size, Serving Unit, and Num Serving."
				showingErr = true
				form.AddFormItem(tview.NewTextView().SetText(errorMsg).SetTextAlign(tview.AlignCenter))
				return
			}
		}

		f.Name = name
		f.BrandName = brandName
		f.Price = price
		f.FoodMacros.Protein = protein
		f.FoodMacros.Carbs = carbs
		f.FoodMacros.Fat = fat
		f.HouseholdServing = hhServing
		f.ServingSize = servingSize
		f.ServingUnit = servingUnit
		f.NumberOfServings = numServings
		f.Calories = bite.CalculateCalories(f.FoodMacros.Protein, f.FoodMacros.Carbs, f.FoodMacros.Fat)

		tx, err := sui.db.Beginx()
		if err != nil {
			log.Printf("couldn't start new transaction: %v\n", err)
			sui.closeModal()
			return
		}
		defer tx.Rollback()

		// Insert food into the foods table.
		f.ID, err = bite.InsertFood(tx, f)
		if err != nil {
			log.Printf("couldn't insert new food: %v\n", err)
			sui.closeModal()
			return
		}

		// Insert food nutrients into the food_nutrients table.
		if err := bite.InsertNutrients(sui.db, tx, f); err != nil {
			log.Printf("failed to insert food nutrients into database: %v\n", err)
			sui.closeModal()
			return
		}

		tx.Commit()
		sui.messages = append(sui.messages, fmt.Sprintf("Created new food %q", f.Name))

		var foods []bite.Food
		text := sui.inputField.GetText()
		switch text == "" {
		case true:
			foods, err = bite.GetRecentlyLoggedFoods(sui.db, bite.SearchLimit)
			if err != nil {
				log.Printf("couldn't get recently logged foods: %v\n", err)
				return
			}
		case false:
			foods = sui.performFoodSearch(text)
		}
		sui.updateFoodsList(foods)

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

// updateSelectedMeal updates the selected meal in the results list.
func (sui *SearchUI) updateSelectedMeal(m bite.Meal) {
	row, col := sui.list.GetSelection()
	cell := sui.list.GetCell(row, col)
	s := "[powderblue]" + m.Name + "[white]"
	cell.SetText(s)
}

// updateSelectedMealFood updates the selected meal food in the results
// list.
func (sui *SearchUI) updateSelectedMealFood(mf bite.MealFood) {
	row, col := sui.list.GetSelection()
	cell := sui.list.GetCell(row, col)
	cell.SetText("* " + mf.Food.Name)
	line := fmt.Sprintf(mfResultsFmt, mf.ServingSize, mf.Food.ServingUnit,
		mf.NumberOfServings, mf.Food.Calories, mf.Food.FoodMacros.Protein,
		mf.Food.FoodMacros.Carbs, mf.Food.FoodMacros.Fat)
	descCell := sui.list.GetCell(row+1, col)
	descCell.SetText(line)
}

func (sui *SearchUI) errorForm(msg string, err error) *tview.Form {
	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle("Error")

	errorTextView := tview.NewTextView().
		SetText(fmt.Sprintf("%s: %v", msg, err)).
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetTextAlign(tview.AlignCenter)
	form.AddFormItem(errorTextView)

	form.AddButton("Ok", func() {
		// Close the form when the "Ok" button is clicked
		sui.closeModal()
	})

	return form
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
