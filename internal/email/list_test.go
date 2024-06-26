package email

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/harryzcy/mailbox/internal/api"
	"github.com/stretchr/testify/assert"
)

func TestList(t *testing.T) {
	tests := []struct {
		client      func(t *testing.T) api.QueryAPI
		now         func() time.Time
		input       ListInput
		expected    *ListResult
		expectedErr error
	}{
		{
			client: func(t *testing.T) api.QueryAPI {
				t.Helper()
				return mockQueryAPI(func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
					return &dynamodb.QueryOutput{
						Items: []map[string]dynamodbTypes.AttributeValue{
							{
								"MessageID":     &dynamodbTypes.AttributeValueMemberS{Value: "exampleMessageID"},
								"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "inbox#2022-03"},
								"DateTime":      &dynamodbTypes.AttributeValueMemberS{Value: "12-01:01:01"},
							},
						},
						LastEvaluatedKey: map[string]dynamodbTypes.AttributeValue{
							"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: "exampleMessageID"},
						},
					}, nil
				})
			},
			input: ListInput{
				Type:  "inbox",
				Year:  "2022",
				Month: "03",
				Order: "desc",
				NextCursor: &Cursor{
					QueryInfo: QueryInfo{
						Type:  "inbox",
						Year:  "2022",
						Month: "03",
						Order: "desc",
					},
				},
			},
			expected: &ListResult{
				Count: 1,
				Items: []Item{
					{
						TimeIndex: TimeIndex{
							MessageID:    "exampleMessageID",
							Type:         "inbox",
							TimeReceived: "2022-03-12T01:01:01Z",
						},
						Unread: new(bool),
					},
				},
				NextCursor: &Cursor{
					QueryInfo: QueryInfo{
						Type:  "inbox",
						Year:  "2022",
						Month: "03",
						Order: "desc",
					},
					LastEvaluatedKey: map[string]dynamodbTypes.AttributeValue{
						"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: "exampleMessageID"},
					},
				},
				HasMore: true,
			},
		},
		{
			client: func(t *testing.T) api.QueryAPI {
				t.Helper()
				return mockQueryAPI(func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
					return &dynamodb.QueryOutput{
						Items: []map[string]dynamodbTypes.AttributeValue{
							{
								"MessageID":     &dynamodbTypes.AttributeValueMemberS{Value: "exampleMessageID"},
								"TypeYearMonth": &dynamodbTypes.AttributeValueMemberS{Value: "inbox#2022-03"},
								"DateTime":      &dynamodbTypes.AttributeValueMemberS{Value: "12-01:01:01"},
							},
						},
						LastEvaluatedKey: map[string]dynamodbTypes.AttributeValue{
							"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: "exampleMessageID"},
						},
					}, nil
				})
			},
			now: func() time.Time { return time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC) },
			input: ListInput{
				Type: "inbox",
			},
			expected: &ListResult{
				Count: 1,
				Items: []Item{
					{
						TimeIndex: TimeIndex{
							MessageID:    "exampleMessageID",
							Type:         "inbox",
							TimeReceived: "2022-03-12T01:01:01Z",
						},
						Unread: new(bool),
					},
				},
				NextCursor: &Cursor{
					QueryInfo: QueryInfo{
						Type:  "inbox",
						Year:  "2022",
						Month: "03",
						Order: "desc",
					},
					LastEvaluatedKey: map[string]dynamodbTypes.AttributeValue{
						"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: "exampleMessageID"},
					},
				},
				HasMore: true,
			},
		},
		{
			client: func(t *testing.T) api.QueryAPI {
				t.Helper()
				return mockQueryAPI(func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
					return &dynamodb.QueryOutput{}, nil
				})
			},
			input: ListInput{
				Type:      "inbox",
				ShowTrash: "error",
			},
			expectedErr: api.ErrInvalidInput,
		},
		{
			client: func(t *testing.T) api.QueryAPI {
				t.Helper()
				return mockQueryAPI(func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
					assert.Fail(t, "this shouldn't be reached")
					return &dynamodb.QueryOutput{}, nil
				})
			},
			input: ListInput{
				Type: "invalid",
			},
			expectedErr: api.ErrInvalidInput,
		},
		{
			client: func(t *testing.T) api.QueryAPI {
				t.Helper()
				return mockQueryAPI(func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
					assert.Fail(t, "this shouldn't be reached")
					return &dynamodb.QueryOutput{}, nil
				})
			},
			input: ListInput{
				Type: "sent",
				Year: "0",
			},
			expectedErr: api.ErrInvalidInput,
		},
		{
			client: func(t *testing.T) api.QueryAPI {
				t.Helper()
				return mockQueryAPI(func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
					return &dynamodb.QueryOutput{}, errors.New("error")
				})
			},
			input: ListInput{
				Type:  "draft",
				Year:  "2022",
				Month: "3",
			},
			expectedErr: errors.New("error"),
		},
		{
			client: func(t *testing.T) api.QueryAPI {
				t.Helper()
				return mockQueryAPI(func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
					return &dynamodb.QueryOutput{}, nil
				})
			},
			input: ListInput{
				Type: "draft",
				NextCursor: &Cursor{
					QueryInfo: QueryInfo{
						Type: "inbox",
					},
					LastEvaluatedKey: map[string]dynamodbTypes.AttributeValue{
						"MessageID": &dynamodbTypes.AttributeValueMemberS{Value: "exampleMessageID"},
					},
				},
			},
			expectedErr: api.ErrQueryNotMatch,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.TODO()
			if test.now != nil {
				now = test.now
			}
			actual, err := List(ctx, test.client(t), test.input)
			assert.Equal(t, test.expected, actual)
			assert.Equal(t, test.expectedErr, err)
		})
	}

	defer func() {
		now = time.Now // cleanup
	}()
}

func TestGetCurrentYearMonth(t *testing.T) {
	tests := []struct {
		now           time.Time
		expectedYear  string
		expectedMonth string
	}{
		{
			now:           time.Date(2022, 3, 3, 0, 0, 0, 0, time.UTC),
			expectedYear:  "2022",
			expectedMonth: "03",
		},
		{
			now:           time.Date(2022, 12, 3, 0, 0, 0, 0, time.UTC),
			expectedYear:  "2022",
			expectedMonth: "12",
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			now = func() time.Time {
				return test.now
			}

			year, month := getCurrentYearMonth()
			assert.Equal(t, test.expectedYear, year)
			assert.Equal(t, test.expectedMonth, month)
			assert.Len(t, year, 4)
			assert.Len(t, month, 2)
		})
	}

	defer func() {
		now = time.Now // cleanup
	}()
}

func TestPrepareYearMonth(t *testing.T) {
	tests := []struct {
		yearIn      string
		monthIn     string
		yearOut     string
		monthOut    string
		expectedErr error
	}{
		{
			yearIn:      "2022",
			yearOut:     "2022",
			monthIn:     "3",
			monthOut:    "03",
			expectedErr: nil,
		},
		{
			yearIn:      "2020",
			yearOut:     "2020",
			monthIn:     "1",
			monthOut:    "01",
			expectedErr: nil,
		},
		{
			yearIn:      "999",
			monthIn:     "1",
			expectedErr: api.ErrInvalidInput,
		},
		{
			yearIn:      "2022",
			monthIn:     "0",
			expectedErr: api.ErrInvalidInput,
		},
		{
			yearIn:      "2022",
			monthIn:     "13",
			expectedErr: api.ErrInvalidInput,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			year, month, err := prepareYearMonth(test.yearIn, test.monthIn)
			assert.Equal(t, test.yearOut, year)
			assert.Equal(t, test.monthOut, month)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
