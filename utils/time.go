package utils

import (
	"dfms/pkg/constants"
	"fmt"
	"time"
)

// ParseDate parses DD/MM/YYYY (list filters and Sage-style dates) as UTC midnight.
func ParseDate(dateString string) (time.Time, error) {
	if dateString == "" {
		return time.Time{}, fmt.Errorf("%w: date string is empty", constants.ErrInvalidInput)
	}

	// Go's reference time: 02/01/2006 = 2nd January, 2006
	t, err := time.Parse("02/01/2006", dateString)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format (expected DD/MM/YYYY): %w", err)
	}
	return t, nil
}
