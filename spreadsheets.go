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
	refreshTokenURL = "https://www.googleapis.com/oauth2/v4/token"
	spreadsheetURL  = "https://sheets.googleapis.com/v4/spreadsheets"
)

// RefreshGoogleToken will pass refreshToken to Google to get the Access Token
func RefreshGoogleToken(refreshToken, clientID, clientSecret string) string {

	type RefreshTokenResponse struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	request := gorequest.New()

	resp, body, errs := request.Post(refreshTokenURL).
		Param("refresh_token", refreshToken).
		Param("client_id", clientID).
		Param("client_secret", clientSecret).
		Param("grant_type", "refresh_token").
		End()

	CheckErrs(errs)

	if resp.StatusCode == 200 {
		var successfulResponse RefreshTokenResponse
		byteArray := []byte(body)
		err := json.Unmarshal(byteArray, &successfulResponse)
		CheckErr(err)
		return successfulResponse.AccessToken
	}

	CheckErr(errors.New(fmt.Sprintf("RefreshGoogleToken. Status Code: %d", resp.StatusCode)))

	return ""

}

// RequestSheetValues will get the resources from Spreadsheet's API
func RequestSheetValues(token, spreadsheetID, sheetID string) [][]string {

	type SheetValues struct {
		Values [][]string `json:"values"`
	}

	request := gorequest.New()

	resp, body, errs := request.Get(fmt.Sprintf("%s/%s/values/%s",
		spreadsheetURL, spreadsheetID, sheetID)).
		Set("Authorization", "Bearer "+token).
		End()

	CheckErrs(errs)

	if resp.StatusCode == 200 {
		var values SheetValues
		byteArray := []byte(body)
		err := json.Unmarshal(byteArray, &values)
		CheckErr(err)
		return values.Values
	}

	CheckErr(errors.New(fmt.Sprintf("RequestSheetValues. Status Code: ", resp.StatusCode)))

	return [][]string{[]string{""}}
}

// PutSheetValues will update the cells in the spreadsheet in a given range
func PutSheetValues(row []string, sheetRange, token, spreadsheetID string) {

	type putDataSchema struct {
		Values         [1][]string `json:"values"`
		Range          string      `json:"range"`
		MajorDimension string      `json:"majorDimension"`
	}

	putData := putDataSchema{Values: [1][]string{row}, Range: sheetRange, MajorDimension: "ROWS"}

	jsonData, err := json.Marshal(putData)

	CheckErr(err)

	request := gorequest.New()

	_, _, errs := request.Put(fmt.Sprintf("%s/%s/values/%s", spreadsheetURL, spreadsheetID, sheetRange)).
		Set("Authorization", "Bearer "+token).
		Param("valueInputOption", "RAW").
		Send(string(jsonData)).
		End()

	CheckErrs(errs)
}

// StoreResults will append a row to your Spreadsheet resource with the day's results.
func StoreResults(token, spreadsheetID, frequency string, results []TodoistItem, spreadsheet [][]string) {
	// Create an new array with the same number of columns as a row in the spreadsheet.
	row := make([]string, len(spreadsheet[0]))

	// The first column contains the period.
	row[0] = calculatePeriod(frequency)

	// We search the index of each item and assign result to the determined column.
	for _, item := range results {
		columnIndex := StringIndexOf(spreadsheet[0], item.Content)
		if item.Checked == 1 {
			row[columnIndex] = "pass"
		} else {
			row[columnIndex] = "fail"
		}
	}

	rowIndex := len(spreadsheet)
	// If the period of the last row of the spreadsheet is different to today's period, our produced row will be new and appended.
	if spreadsheet[len(spreadsheet)-1][0] != row[0] {
		rowIndex = len(spreadsheet) + 1
	}

	// Compose the range
	sheetRange := fmt.Sprintf("%s!%d:%d", frequency, rowIndex, rowIndex)

	// Store the row in the spreadsheet!
	PutSheetValues(row, sheetRange, token, spreadsheetID)
}

// UpdateHabit will update the next iteration of a habit
func UpdateHabit(index int, project []string, token, spreadsheetID string) {
	var interval int
	tomorrow := time.Now().AddDate(0, 0, 1)

	if project[2] == "" {
		interval = 1
	} else {
		_interval, err := strconv.Atoi(project[2])
		CheckErr(err)
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

	project[4] = DateToString(nextIteration)
	sheetRange := fmt.Sprintf("Habits!%d:%d", index+1, index+1)
	PutSheetValues(project, sheetRange, token, spreadsheetID)
}

func calculatePeriod(frequency string) string {
	today := time.Now()
	year := today.Year()
	switch {
	case frequency == "day":
		return DateToString(today)
	case frequency == "week":
		_, week := today.ISOWeek()
		return fmt.Sprintf("%d %d", week, year)
	case frequency == "month":
		return fmt.Sprintf("%s %d", today.Month(), year)
	}
	return fmt.Sprintf("%d", year)
}
