package email

import (
	"context"
	"strconv"
	"time"
)

// ListInput represents the input of list method
type ListInput struct {
	Type       string `json:"type"`
	Year       string `json:"year"`
	Month      string `json:"month"`
	Order      string `json:"order"` // asc or desc (default)
	NextCursor Cursor `json:"nextCursor"`
}

// ListResult represents the result of list method
type ListResult struct {
	Count      int         `json:"count"`
	Items      []TimeIndex `json:"items"`
	NextCursor string      `json:"nextCursor"`
}

// List lists emails in DynamoDB
func List(ctx context.Context, api QueryAPI, input ListInput) (*ListResult, error) {
	if input.Type != EmailTypeInbox && input.Type != EmailTypeDraft && input.Type != EmailTypeSent {
		return nil, ErrInvalidInput
	}

	if input.Year == "" && input.Month == "" {
		input.Year, input.Month = getCurrentYearMonth()
		input.Order = "desc"
	} else {
		var err error
		input.Year, input.Month, err = prepareYearMonth(input.Year, input.Month)
		if err != nil {
			return nil, err
		}
		if input.Order == "" {
			input.Order = "desc"
		}
	}

	result, err := listByYearMonth(ctx, api, listQueryInput{
		emailType:        input.Type,
		year:             input.Year,
		month:            input.Month,
		order:            input.Order,
		lastEvaluatedKey: nil,
	})
	if err != nil {
		return nil, err
	}

	return &ListResult{
		Count: len(result.items),
		Items: result.items,
	}, nil
}

func getCurrentYearMonth() (year, month string) {
	now := time.Now().UTC()

	year = strconv.Itoa(now.Year())
	month = strconv.Itoa(int(now.Month()))

	return year, month
}

// prepareYearMonth ensures year and month are valid
// and returns 4 digit year snd 2 digit month
func prepareYearMonth(year string, month string) (string, string, error) {
	if len(month) == 1 {
		month = "0" + month
	}

	// Year is 4 digit number string
	if yearNum, err := strconv.Atoi(year); err != nil || yearNum < 1000 {
		return "", "", ErrInvalidInput
	}
	// Month is 2 digit number string
	if monthNum, err := strconv.Atoi(month); err != nil || monthNum > 12 || monthNum < 1 {
		return "", "", ErrInvalidInput
	}
	return year, month, nil
}
