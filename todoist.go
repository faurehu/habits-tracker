package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

var (
	todoistAPIURL = "https://todoist.com/API/v7/sync"
)

// TodoistProject represents a Todoist Project
type TodoistProject struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// TodoistItem represents a Todoist Item
type TodoistItem struct {
	ProjectID int    `json:"project_id"`
	Content   string `json:"content"`
	Indent    int    `json:"indent"`
	Checked   int    `json:"checked"`
}

// TodoistGetResourceResponse represents the response from Todoist API
type TodoistGetResourceResponse struct {
	Projects []TodoistProject `json:"projects"`
	Items    []TodoistItem    `json:"items"`
}

type argsSchema struct {
	IDS        []int  `json:"ids"`
	Content    string `json:"content"`
	Name       string `json:"name"`
	Indent     int    `json:"indent"`
	DateString string `json:"date_string"`
	ItemID     string `json:"item_id"`
	ProjectID  string `json:"project_id"`
}

type commandSchema struct {
	Type          string     `json:"type"`
	UUID          string     `json:"uuid"`
	Args          argsSchema `json:"args"`
	TemporaryName string     `json:"temp_id"`
}

// GetResources will fetch the resources from Todoist API
func GetResources(todoistToken string) (TodoistGetResourceResponse, error) {

	form := url.Values{"token": {todoistToken}, "resource_types": {"[\"projects\", \"items\"]"}, "sync_token": {"*"}}

	resp, err := http.PostForm(todoistAPIURL, form)
	if err != nil {
		return TodoistGetResourceResponse{}, errors.Wrap(err, "could not make post")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return TodoistGetResourceResponse{}, fmt.Errorf("Bad status code: %d", resp.StatusCode)
	}

	var tgrr TodoistGetResourceResponse
	if err := json.NewDecoder(resp.Body).Decode(&tgrr); err != nil {
		return TodoistGetResourceResponse{}, errors.Wrap(err, "could not decode JSON")
	}

	return tgrr, nil
}

// DeleteProject will destroy the project with given ID
func DeleteProject(ID int, todoistToken string) error {

	args := argsSchema{IDS: []int{ID}}
	commands := []commandSchema{commandSchema{Type: "project_delete", UUID: "random_string", Args: args, TemporaryName: "deleted_project"}}

	return postCommands(commands, todoistToken)
}

// CreateProject will create a Project and return it's id
func CreateProject(todoistToken string) error {

	projectAddArgs := argsSchema{Name: "Habits", Indent: 1}
	createProject := commandSchema{Type: "project_add", UUID: getUUID(0), Args: projectAddArgs, TemporaryName: "project_id"}

	firstIndentArgs := argsSchema{Content: time.Now().AddDate(0, 0, 1).Format(DateFormat), Indent: 1, ProjectID: "project_id"}
	createFirstIndent := commandSchema{Type: "item_add", UUID: getUUID(1), Args: firstIndentArgs, TemporaryName: "first_indent"}

	commands := []commandSchema{createProject, createFirstIndent}
	err := postCommands(commands, todoistToken)
	return err
}

func findHabitProject(todoistToken string) (string, error) {
	todoistResponse, err := GetResources(todoistToken)
	if err != nil {
		return "", errors.Wrap(err, "could not fetch Todoist API data")
	}

	for _, project := range todoistResponse.Projects {
		if project.Name == "Habits" {
			return strconv.Itoa(project.ID), nil
		}
	}

	return "", fmt.Errorf("could not find Habit project")
}

// CreateHabitTasks posts the day's tasks to Todoist
func CreateHabitTasks(programmedHabits [][]string, todoistToken string) error {

	err := CreateProject(todoistToken)
	if err != nil {
		return err
	}

	projectID, err := findHabitProject(todoistToken)
	if err != nil {
		return err
	}

	for index, habit := range programmedHabits {
		habitID := fmt.Sprintf("habit-%d", index)

		habitArgs := argsSchema{Content: habit[0], Indent: 2, ProjectID: projectID}
		addHabitCommand := commandSchema{Type: "item_add", UUID: getUUID((index + 2) * 2), Args: habitArgs, TemporaryName: habitID}
		habitCommands := []commandSchema{addHabitCommand}

		if habit[3] != "" {
			reminderArgs := argsSchema{DateString: fmt.Sprintf("tomorrow at %s", habit[3]), ItemID: habitID}
			reminderCommand := commandSchema{Type: "reminder_add", UUID: getUUID((index+2)*2 + 1), Args: reminderArgs, TemporaryName: habitID + "r"}
			habitCommands = append(habitCommands, reminderCommand)
		}

		err := postCommands(habitCommands, todoistToken)
		if err != nil {
			return err
		}
	}

	return nil
}

func postCommands(commands []commandSchema, todoistToken string) error {

	commandsJSON, err := json.Marshal(commands)
	if err != nil {
		return errors.Wrap(err, "could not marshal command data")
	}

	r, err := http.NewRequest("POST", todoistAPIURL, nil)
	if err != nil {
		return errors.Wrap(err, "could not construct a request")
	}

	q := r.URL.Query()
	q.Add("token", todoistToken)
	q.Add("commands", string(commandsJSON))
	r.URL.RawQuery = q.Encode()

	client := http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		return errors.Wrap(err, "could not make request")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Bad status code: %d", resp.StatusCode)
	}

	return nil
}

func getUUID(index int) string {
	return fmt.Sprintf("%d-%d", time.Now().Unix(), index)
}
