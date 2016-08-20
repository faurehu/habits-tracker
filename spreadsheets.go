package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

var (
	refreshTokenURL = "https://www.googleapis.com/oauth2/v4/token"
	spreadsheetURL  = "https://sheets.googleapis.com/v4/spreadsheets"
)

// RefreshGoogleToken will pass refreshToken to Google to get the Access Token
func RefreshGoogleToken(refreshToken, clientID, clientSecret string) (string, error) {

	type RefreshTokenResponse struct {
		AccessToken string `json:"access_token"`
	}

	form := url.Values{"refresh_token": {refreshToken}, "client_id": {clientID}, "client_secret": {clientSecret}, "grant_type": {"refresh_token"}}

	resp, err := http.PostForm(refreshTokenURL, form)
	if err != nil {
		return "", errors.Wrap(err, "could not make request")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Bad status code: %d", resp.StatusCode)
	}

	var rtr RefreshTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&rtr); err != nil {
		return "", errors.Wrap(err, "could not decode response body")
	}

	return rtr.AccessToken, nil
}

// RequestSheetValues will get the resources from Spreadsheet's API
func RequestSheetValues(token, spreadsheetID, sheetID string) ([][]string, error) {

	type SheetValues struct {
		Values [][]string `json:"values"`
	}

	cookedURL := fmt.Sprintf("%s/%s/values/%s", spreadsheetURL, spreadsheetID, sheetID)

	r, err := http.NewRequest("GET", cookedURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not build request")
	}
	r.Header.Set("Authorization", "Bearer "+token)

	client := http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		return nil, errors.Wrap(err, "could not make request")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Bad status code: %d", resp.StatusCode)
	}

	var v SheetValues
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, errors.Wrap(err, "could not decode response body")
	}

	return v.Values, nil
}

// PutSheetValues will update the cells in the spreadsheet in a given range
func PutSheetValues(row []string, sheetRange, token, spreadsheetID string) error {

	type putDataSchema struct {
		Values         [1][]string `json:"values"`
		Range          string      `json:"range"`
		MajorDimension string      `json:"majorDimension"`
	}

	putData := putDataSchema{Values: [1][]string{row}, Range: sheetRange, MajorDimension: "ROWS"}

	jsonData, err := json.Marshal(putData)
	if err != nil {
		return errors.Wrap(err, "could not marshal data")
	}

	cookedURL := fmt.Sprintf("%s/%s/values/%s", spreadsheetURL, spreadsheetID, sheetRange)

	r, err := http.NewRequest("PUT", cookedURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return errors.Wrap(err, "could not build request")
	}
	r.Header.Set("Authorization", "Bearer "+token)
	q := r.URL.Query()
	q.Add("valueInputOption", "RAW")
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

// StoreResults will append a row to your Spreadsheet resource with the day's results.
func StoreResults(token, spreadsheetID, frequency string, results []TodoistItem, spreadsheet [][]string) error {
	// Create an new array with the same number of columns as a row in the spreadsheet.
	row := make([]string, len(spreadsheet[0]))

	// The first column contains the period.
	row[0] = calculatePeriod(frequency)

	// We search the index of each item and assign result to the determined column.
	for _, item := range results {
		columnIndex := stringIndexOf(spreadsheet[0], item.Content)
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
	err := PutSheetValues(row, sheetRange, token, spreadsheetID)
	if err != nil {
		return errors.Wrap(err, "could not send data to Spreadsheets API")
	}

	return nil
}

// UpdateHabit will update the next iteration of a habit
func UpdateHabit(index int, project []string, token, spreadsheetID string) error {
	var interval int
	tomorrow := time.Now().AddDate(0, 0, 1)

	if project[2] == "" {
		interval = 1
	} else {
		_interval, err := strconv.Atoi(project[2])
		if err != nil {
			return errors.Wrap(err, "could not convert sheet data to integer")
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

	project[4] = nextIteration.Format(DateFormat)
	sheetRange := fmt.Sprintf("Habits!%d:%d", index+1, index+1)

	err := PutSheetValues(project, sheetRange, token, spreadsheetID)
	if err != nil {
		return errors.Wrap(err, "could not send data to Spreadsheets API")
	}

	return nil
}

func calculatePeriod(frequency string) string {
	today := time.Now()
	year := today.Year()
	switch {
	case frequency == "day":
		return today.Format(DateFormat)
	case frequency == "week":
		_, week := today.ISOWeek()
		return fmt.Sprintf("%d %d", week, year)
	case frequency == "month":
		return fmt.Sprintf("%s %d", today.Month(), year)
	}
	return fmt.Sprintf("%d", year)
}

func stringIndexOf(slice []string, element string) int {
	for index, item := range slice {
		if item == element {
			return index
		}
	}
	return -1
}
