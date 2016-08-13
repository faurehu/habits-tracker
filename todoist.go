package habits

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/parnurzeal/gorequest"
	"strconv"
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
func GetResources(todoistToken string) (TodoistGetResourceResponse, []error) {

	request := gorequest.New()

	resp, body, errs := request.Get(todoistAPIURL).
		Param("token", todoistToken).
		Param("resource_types", "[\"projects\", \"items\"]").
		Param("sync_token", "*").
		End()

	CheckErrs(errs)

	if resp.StatusCode == 200 {
		var todoistResponse TodoistGetResourceResponse
		byteArray := []byte(body)
		err := json.Unmarshal(byteArray, &todoistResponse)
		CheckErr(err)
		return todoistResponse, nil
	}

	return TodoistGetResourceResponse{}, []error{errors.New("Response not OK. Status Code: " + strconv.Itoa(resp.StatusCode))}

}

// DeleteProject will destroy the project with given ID
func DeleteProject(ID int, todoistToken string) {

	args := argsSchema{IDS: []int{ID}}
	commands := []commandSchema{commandSchema{Type: "project_delete", UUID: "random_string", Args: args}}
	commandsJSON, err := json.Marshal(commands)

	CheckErr(err)

	request := gorequest.New()

	_, _, errs := request.Post(todoistAPIURL).
		Param("token", todoistToken).
		Param("commands", string(commandsJSON)).
		End()

	CheckErrs(errs)
}

// CreateHabitTasks posts the day's tasks to Todoist
func CreateHabitTasks(programmedHabits [][]string, todoistToken string) {

	projectAddArgs := argsSchema{Name: "Habits", Indent: 1}
	createProject := commandSchema{Type: "project_add", UUID: getUUID(0), Args: projectAddArgs, TemporaryName: "habits_project"}

	firstIndentArgs := argsSchema{Content: DateToString(time.Now().AddDate(0, 0, 1)), Indent: 1, ProjectID: "habits_project"}
	createFirstIndent := commandSchema{Type: "item_add", UUID: getUUID(1), Args: firstIndentArgs, TemporaryName: "firstindent"}

	commands := []commandSchema{createProject, createFirstIndent}
	for index, habit := range programmedHabits {
		habitID := fmt.Sprintf("habit%d", index)
		habitArgs := argsSchema{Content: habit[0], Indent: 2, ProjectID: "habits_project"}
		addHabitCommand := commandSchema{Type: "item_add", UUID: getUUID((index + 2) * 2), Args: habitArgs, TemporaryName: habitID}
		commands = append(commands, addHabitCommand)

		if habit[3] != "" {
			reminderArgs := argsSchema{DateString: habit[3], ItemID: habitID}
			reminderCommand := commandSchema{Type: "reminder_add", UUID: getUUID((index+2)*2 + 1), Args: reminderArgs}
			commands = append(commands, reminderCommand)
		}
	}

	commandsJSON, err := json.Marshal(commands)
	jsonString := string(commandsJSON)

	CheckErr(err)

	request := gorequest.New()

	_, _, errs := request.Post(todoistAPIURL).
		Param("token", todoistToken).
		Param("commands", jsonString).
		End()

	CheckErrs(errs)
}

func getUUID(index int) string {
	return fmt.Sprintf("%d-%d", time.Now().Unix(), index)
}
