package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

// Date Format is the format for the dates layout
const DateFormat = "2 January 2006"

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

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load the configuration
	config, err := loadConfig()
	if err != nil {
		return errors.Wrap(err, "could not load configuration")
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
	nextIteration := make([]string, len(habitsMainSpreadsheet))

	for index, project := range habitsMainSpreadsheet {
		if tomorrow == project[4] {
			programmedHabits = append(programmedHabits, project)

			// We calculate the next iteration
			calculatedIteration, err := calculateNextIteration(project)
			if err != nil {
				return errors.Wrap(err, "could not calculate next iteration")
			}

			nextIteration[index] = calculatedIteration
		}
	}

	// We send the next iterations column to the habits database
	err = PutSheetValues(nextIteration, "Habits!E:E", "COLUMNS", token, config.SpreadsheetID)
	if err != nil {
		return errors.Wrap(err, "could not update habits in the Sheets API")
	}

	// Finally, send these to Todoist
	err = CreateHabitTasks(programmedHabits, config.TodoistToken)
	if err != nil {
		return errors.Wrap(err, "could not send habits to Todoist API")
	}

	return nil
}

func loadConfig() (Configuration, error) {
	file, err := os.Open("config.json")
	if err != nil {
		return Configuration{}, errors.Wrap(err, "couldn't open config file")
	}

	var config Configuration
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return Configuration{}, errors.Wrap(err, "could not decode config file content")
	}
	return config, nil
}

func calculateNextIteration(project []string) (string, error) {
	var interval int
	tomorrow := time.Now().AddDate(0, 0, 1)

	if project[2] == "" {
		interval = 1
	} else {
		_interval, err := strconv.Atoi(project[2])
		if err != nil {
			return "", errors.Wrap(err, "could not convert sheet data to integer")
		}

		interval = _interval
	}

	var nextIteration time.Time
	switch {
	case project[1] == "day":
		nextIteration = tomorrow.AddDate(0, 0, interval)
	case project[1] == "week":
		nextIteration = tomorrow.AddDate(0, 0, interval*7)
	case project[1] == "month":
		nextIteration = tomorrow.AddDate(0, interval, 0)
	case project[1] == "year":
		nextIteration = tomorrow.AddDate(interval, 0, 0)
	}

	return nextIteration.Format(DateFormat), nil
}
