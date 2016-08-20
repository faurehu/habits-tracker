package main

import (
	"encoding/json"
	"github.com/faurehu/habittracker"
	"os"
	"time"
)

type Frequency struct {
	Spreadsheet [][]string
	Results     []habittracker.TodoistItem
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
	habittracker.CheckErr(err)
	decoder := json.NewDecoder(file)
	config := Configuration{}
	err = decoder.Decode(&config)
	habittracker.CheckErr(err)

	// First of all, refresh Google Token.
	token, err := habittracker.RefreshGoogleToken(config.RefreshToken, config.ClientID, config.ClientSecret)

	habittracker.CheckErr(err)

	// Let's load our current database.
	habitsMainSpreadsheet, err := habittracker.RequestSheetValues(token, config.SpreadsheetID, "Habits")

	habittracker.CheckErr(err)

	// A map of key frequency and values results and spreadsheet will help keep everything tidy
	frequencies := [4]string{"day", "week", "month", "year"}
	frequencyMap := map[string]Frequency{}
	for _, frequency := range frequencies {
		spreadsheet, err := habittracker.RequestSheetValues(token, config.SpreadsheetID, frequency)
		habittracker.CheckErr(err)
		frequencyMap[frequency] = Frequency{Spreadsheet: spreadsheet, Results: []habittracker.TodoistItem{}}
	}

	// A map of habit name: habit row will be useful, let's build it.
	habitMap := map[string][]string{}
	for _, habitRow := range habitsMainSpreadsheet[1:] {
		habitMap[habitRow[0]] = habitRow
	}

	// It's 11:30pm, let's load the results from Todoist. Each slice contains items of their respective frequency.

	// First, we get all of the items and projects from Todoist.
	todoistResponse, err := habittracker.GetResources(config.TodoistToken)
	habittracker.CheckErr(err)

	// Find the Habits project.
	var habitsProject habittracker.TodoistProject
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
			err := habittracker.StoreResults(token, config.SpreadsheetID, frequency, frequencyRecord.Results, frequencyRecord.Spreadsheet)
			habittracker.CheckErr(err)
		}
	}

	// At this point we can safely remove today's items from Todoist.
	err = habittracker.DeleteProject(habitsProject.ID, config.TodoistToken)
	habittracker.CheckErr(err)

	// Build tomorrow's items
	programmedHabits := [][]string{}

	// We will need tomorrow's dates
	tomorrow := time.Now().AddDate(0, 0, 1).Format(habittracker.DateFormat)
	for index, project := range habitsMainSpreadsheet {
		if tomorrow == project[4] {
			programmedHabits = append(programmedHabits, project)
			err = habittracker.UpdateHabit(index, project, token, config.SpreadsheetID)
			habittracker.CheckErr(err)
		}
	}

	// Finally, send these to Todoist
	err = habittracker.CreateHabitTasks(programmedHabits, config.TodoistToken)
	habittracker.CheckErr(err)
}
