package habittracker

// DateFormat to convert dates to strings
const DateFormat = "2 January 2006"

// Errors struct to implement the error interface.
type Errors struct {
	ErrorItems []error
}

// Implements the Error function of the error interface.
func (e Errors) Error() string {
	errorMessage := ""
	for _, item := range e.ErrorItems {
		errorMessage = errorMessage + " " + item.Error()
	}
	return errorMessage
}

// CheckErr will handle errors.
func CheckErr(err error) {
	if err != nil {
		panic(err)
	}
}

// CheckErrs will handle slices of errors.
func CheckErrs(errs []error) {
	if errs != nil {
		errors := Errors{ErrorItems: errs}
		panic(errors)
	}
}

// StringIndexOf will return the index of a specified string in a given slice.
func StringIndexOf(slice []string, element string) int {
	for index, item := range slice {
		if item == element {
			return index
		}
	}
	return -1
}
