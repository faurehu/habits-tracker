package main

import (
	"encoding/json"
	"habits"
	"os"
	"time"
)

type Frequency struct {
	Spreadsheet [][]string
	Results     []habits.TodoistItem
}

type Configuration struct {
	SpreadsheetID string
	RefreshToken  string
	ClientID      string
	ClientSecret  string
	TodoistToken  string
}

func main() {
	// Load the configuration
	file, err := os.Open("config.json")
	habits.CheckErr(err)
	decoder := json.NewDecoder(file)
	config := Configuration{}
	err = decoder.Decode(&config)
	habits.CheckErr(err)

	// First of all, refresh Google Token.
	token := habits.RefreshGoogleToken(config.RefreshToken, config.ClientID, config.ClientSecret)

	// Let's load our current database.
	habitsMainSpreadsheet := habits.RequestSheetValues(token, config.SpreadsheetID, "Habits")

	// A map of key frequency and values results and spreadsheet will help keep everything tidy
	frequencies := [4]string{"day", "week", "month", "year"}
	frequencyMap := map[string]Frequency{}
	for _, frequency := range frequencies {
		frequencyMap[frequency] = Frequency{Spreadsheet: habits.RequestSheetValues(token, config.SpreadsheetID, frequency), Results: []habits.TodoistItem{}}
	}

	// A map of habit name: habit row will be useful, let's build it.
	habitMap := map[string][]string{}
	for _, habitRow := range habitsMainSpreadsheet[1:] {
		habitMap[habitRow[0]] = habitRow
	}

	// It's 11:30pm, let's load the results from Todoist. Each slice contains items of their respective frequency.

	// First, we get all of the items and projects from Todoist.
	todoistResponse, errs := habits.GetResources(config.TodoistToken)
	habits.CheckErrs(errs)

	// Find the Habits project.
	var habitsProject habits.TodoistProject
	for _, project := range todoistResponse.Projects {
		if project.Name == "Habits" {
			habitsProject = project
			break
		}
	}

	// Sort the items to their respective slice.
	for _, item := range todoistResponse.Items {
		if item.ProjectID == habitsProject.ID && item.Indent == 2 {
			frequencyName := habitMap[item.Content][1]
			frequency := frequencyMap[frequencyName]
			frequency.Results = append(frequency.Results, item)
			frequencyMap[frequencyName] = frequency
		}
	}

	// Only update the non-daily spreadsheets if there are items to update.
	for _, frequency := range frequencies {
		frequencyRecord := frequencyMap[frequency]
		if len(frequencyRecord.Results) > 0 {
			habits.StoreResults(token, config.SpreadsheetID, frequency, frequencyRecord.Results, frequencyRecord.Spreadsheet)
		}
	}

	// At this point we can safely remove today's items from Todoist.
	habits.DeleteProject(habitsProject.ID, config.TodoistToken)

	// Build tomorrow's items
	programmedHabits := [][]string{}

	// We will need tomorrow's dates
	tomorrow := time.Now().AddDate(0, 0, 1).Format(habits.DateFormat)
	for index, project := range habitsMainSpreadsheet {
		if tomorrow == project[4] {
			programmedHabits = append(programmedHabits, project)
			habits.UpdateHabit(index, project, token, config.SpreadsheetID)
		}
	}

	// Finally, send these to Todoist
	habits.CreateHabitTasks(programmedHabits, config.TodoistToken)
}
