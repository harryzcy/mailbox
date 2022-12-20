package email

import (
	"context"
	"strconv"
	"time"
)

const (
	DEFAULT_PAGE_SIZE = 100
)

// ListInput represents the input of list method
type ListInput struct {
	Type       string  `json:"type"`
	Year       string  `json:"year"`
	Month      string  `json:"month"`
	Order      string  `json:"order"`    // asc or desc (default)
	PageSize   int     `json:"pageSize"` // 0 means no limit, default is 100
	NextCursor *Cursor `json:"nextCursor"`
}

// ListResult represents the result of list method
type ListResult struct {
	Count      int         `json:"count"`
	Items      []TimeIndex `json:"items"`
	NextCursor *Cursor     `json:"nextCursor"`
	HasMore    bool        `json:"hasMore"`
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

	inputs := listQueryInput{
		emailType: input.Type,
		year:      input.Year,
		month:     input.Month,
		order:     input.Order,
		pageSize:  input.PageSize,
	}

	if input.NextCursor != nil && len(input.NextCursor.LastEvaluatedKey) > 0 {
		if input.NextCursor.QueryInfo.Type != input.Type ||
			input.NextCursor.QueryInfo.Year != input.Year || input.NextCursor.QueryInfo.Month != input.Month ||
			input.NextCursor.QueryInfo.Order != input.Order {
			return nil, ErrQueryNotMatch
		}

		inputs.lastEvaluatedKey = input.NextCursor.LastEvaluatedKey
	}
	result, err := listByYearMonth(ctx, api, inputs)
	if err != nil {
		return nil, err
	}

	var nextCursor *Cursor
	if result.hasMore {
		nextCursor = &Cursor{
			QueryInfo: QueryInfo{
				Type:  input.Type,
				Year:  input.Year,
				Month: input.Month,
				Order: input.Order,
			},
			LastEvaluatedKey: result.lastEvaluatedKey,
		}
	}

	return &ListResult{
		Count:      len(result.items),
		Items:      result.items,
		NextCursor: nextCursor,
		HasMore:    result.hasMore,
	}, nil
}

// now is equal to time.Now, but will be replaced during testing
var now = time.Now

func getCurrentYearMonth() (year, month string) {
	now := now().UTC()

	year = strconv.Itoa(now.Year())
	month = strconv.Itoa(int(now.Month()))
	if len(month) == 1 {
		month = "0" + month
	}

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
