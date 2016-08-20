package habits

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
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
		return TodoistGetResourceResponse{}, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return TodoistGetResourceResponse{}, fmt.Errorf("Bad status code: %d", resp.StatusCode)
	}

	var tgrr TodoistGetResourceResponse
	if err := json.NewDecoder(resp.Body).Decode(&tgrr); err != nil {
		return TodoistGetResourceResponse{}, err
	}

	return tgrr, nil
}

// DeleteProject will destroy the project with given ID
func DeleteProject(ID int, todoistToken string) error {

	args := argsSchema{IDS: []int{ID}}
	commands := []commandSchema{commandSchema{Type: "project_delete", UUID: "random_string", Args: args, TemporaryName: "deletedproject"}}
	commandsJSON, err := json.Marshal(commands)

	if err != nil {
		return err
	}

	r, err := http.NewRequest("POST", todoistAPIURL, nil)
	if err != nil {
		return err
	}
	q := r.URL.Query()
	q.Add("token", todoistToken)
	q.Add("commands", string(commandsJSON))
	r.URL.RawQuery = q.Encode()

	client := http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Bad status code: %d", resp.StatusCode)
	}

	return nil
}

// CreateHabitTasks posts the day's tasks to Todoist
func CreateHabitTasks(programmedHabits [][]string, todoistToken string) error {

	time.Sleep(5 * time.Second)
	projectAddArgs := argsSchema{Name: "Habits", Indent: 1}
	createProject := commandSchema{Type: "project_add", UUID: getUUID(0), Args: projectAddArgs, TemporaryName: "habits_project"}

	firstIndentArgs := argsSchema{Content: time.Now().AddDate(0, 0, 1).Format(DateFormat), Indent: 1, ProjectID: "habits_project"}
	createFirstIndent := commandSchema{Type: "item_add", UUID: getUUID(1), Args: firstIndentArgs, TemporaryName: "firstindent"}

	commands := []commandSchema{createProject, createFirstIndent}
	for index, habit := range programmedHabits {
		habitID := fmt.Sprintf("habit%d", index)
		habitArgs := argsSchema{Content: habit[0], Indent: 2, ProjectID: "habits_project"}
		addHabitCommand := commandSchema{Type: "item_add", UUID: getUUID((index + 2) * 2), Args: habitArgs, TemporaryName: habitID}
		commands = append(commands, addHabitCommand)

		if habit[3] != "" {
			reminderArgs := argsSchema{DateString: fmt.Sprintf("tomorrow at %s", habit[3]), ItemID: habitID}
			reminderCommand := commandSchema{Type: "reminder_add", UUID: getUUID((index+2)*2 + 1), Args: reminderArgs}
			commands = append(commands, reminderCommand)
		}
	}

	commandsJSON, err := json.Marshal(commands)

	if err != nil {
		return err
	}

	r, err := http.NewRequest("POST", todoistAPIURL, nil)
	if err != nil {
		return err
	}
	q := r.URL.Query()
	q.Add("token", todoistToken)
	q.Add("commands", string(commandsJSON))
	r.URL.RawQuery = q.Encode()

	client := http.Client{}
	resp, err := client.Do(r)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Bad status code: %d", resp.StatusCode)
	}

	return nil
}

func getUUID(index int) string {
	return fmt.Sprintf("%d-%d", time.Now().Unix(), index)
}
