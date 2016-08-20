package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
)

// DateFormat to convert dates to strings
const DateFormat = "2 January 2006"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

type Frequency struct {
	Spreadsheet [][]string
	Results     []TodoistItem
}

type Configuration struct {
	SpreadsheetID string
	RefreshToken  string
	ClientID      string
	ClientSecret  string
	TodoistToken  string
}

func run() error {
	// Load the configuration
	file, err := os.Open("config.json")
	if err != nil {
		return errors.Wrap(err, "couldn't open config file")
	}

	err = decoder.Decode(&config)
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return errors.Wrap(err, "could not decode config file content")
	}

	// First of all, refresh Google Token.
	token, err := RefreshGoogleToken(config.RefreshToken, config.ClientID, config.ClientSecret)
	if err != nil {
		return errors.Wrap(err, "could not refresh Google API token")
	}

	// Let's load our current database.
	habitsMainSpreadsheet, err := RequestSheetValues(token, config.SpreadsheetID, "Habits")
	if err != nil {
		return errors.Wrap(err, "could not fetch Sheets API data")
	}

	// A map of key frequency and values results and spreadsheet will help keep everything tidy
	frequencies := [4]string{"day", "week", "month", "year"}
	frequencyMap := map[string]Frequency{}
	for _, frequency := range frequencies {
		spreadsheet, err := RequestSheetValues(token, config.SpreadsheetID, frequency)
		if err != nil {
			return errors.Wrap(err, "could not fetch Sheets API data")
		}
		frequencyMap[frequency] = Frequency{Spreadsheet: spreadsheet, Results: []TodoistItem{}}
	}

	// A map of habit name: habit row will be useful, let's build it.
	habitMap := map[string][]string{}
	for _, habitRow := range habitsMainSpreadsheet[1:] {
		habitMap[habitRow[0]] = habitRow
	}

	// It's 11:30pm, let's load the results from Todoist. Each slice contains items of their respective frequency.

	// First, we get all of the items and projects from Todoist.
	todoistResponse, err := GetResources(config.TodoistToken)
	if err != nil {
		return errors.Wrap(err, "could not fetch Todoist API data")
	}
	// Find the Habits project.
	var habitsProject TodoistProject
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
			err := StoreResults(token, config.SpreadsheetID, frequency, frequencyRecord.Results, frequencyRecord.Spreadsheet)
			if err != nil {
				return errors.Wrap(err, "could not store results in Sheets API")
			}
		}
	}

	// At this point we can safely remove today's items from Todoist.
	err = DeleteProject(habitsProject.ID, config.TodoistToken)
	if err != nil {
		return errors.Wrap(err, "could not delete Todoist project")
	}
	// Build tomorrow's items
	programmedHabits := [][]string{}

	// We will need tomorrow's dates
	tomorrow := time.Now().AddDate(0, 0, 1).Format(DateFormat)
	for index, project := range habitsMainSpreadsheet {
		if tomorrow == project[4] {
			programmedHabits = append(programmedHabits, project)

			err = UpdateHabit(index, project, token, config.SpreadsheetID)
			if err != nil {
				return errors.Wrap(err, "could not update habits in the Sheets API")
			}
		}
	}

	// Finally, send these to Todoist
	err = CreateHabitTasks(programmedHabits, config.TodoistToken)
	if err != nil {
		return errors.Wrap(err, "could not send habits to Todoist API")
	}

	return nil
}
