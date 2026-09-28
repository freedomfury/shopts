package shopts

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// validate checks one value against e and returns it normalized. Every value
// goes through here: option values, list items, positionals and defaults.
// A value may contain anything but a newline, which would end its line.
func validate(e *entry, v string) (string, error) {
	if strings.ContainsRune(v, '\n') {
		return "", errors.New("must not contain a newline")
	}

	switch e.typ {
	case "int":
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return "", errors.New("must be a valid integer")
		}
		if e.min != nil && n < e.min.i {
			return "", fmt.Errorf("must be at least %s", e.min.raw)
		}
		if e.max != nil && n > e.max.i {
			return "", fmt.Errorf("must be at most %s", e.max.raw)
		}
		v = strconv.FormatInt(n, 10)
	case "float":
		f, err := parseFloat(v)
		if err != nil {
			return "", err
		}
		if e.min != nil && f < e.min.f {
			return "", fmt.Errorf("must be at least %s", e.min.raw)
		}
		if e.max != nil && f > e.max.f {
			return "", fmt.Errorf("must be at most %s", e.max.raw)
		}
	case "bool", "flag":
		b, err := strconv.ParseBool(v)
		if err != nil {
			return "", errors.New("must be true or false")
		}
		v = strconv.FormatBool(b)
	case "enum":
		if !slices.Contains(e.enum, v) {
			return "", fmt.Errorf("must be one of: %s", strings.Join(e.enum, ", "))
		}
	}

	if e.minLength != nil && utf8.RuneCountInString(v) < *e.minLength {
		return "", fmt.Errorf("must be at least %d characters long", *e.minLength)
	}
	if e.maxLength != nil && utf8.RuneCountInString(v) > *e.maxLength {
		return "", fmt.Errorf("must be no more than %d characters long", *e.maxLength)
	}
	if e.match != nil && !e.match.Check(v) {
		return "", errors.New(e.failureMessage())
	}
	return v, nil
}

// failureMessage picks the message for a failed pattern: the option's
// failure=, then the validator's own, then a generic one.
func (e *entry) failureMessage() string {
	switch {
	case e.failure != "":
		return e.failure
	case e.match.Failure != "":
		return e.match.Failure
	default:
		return "must match the pattern " + e.match.source
	}
}

func parseFloat(v string) (float64, error) {
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, errors.New("must be a valid number")
	}
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return 0, errors.New("must be a finite number")
	}
	return f, nil
}
