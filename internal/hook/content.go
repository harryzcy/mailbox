package hook

const (
	EventEmail     = "email"
	ActionReceived = "received"
)

// EmailReceipt contains information needed for an email receipt
type EmailReceipt struct {
	MessageID string
	Timestamp string
}

// EmailNotification contains information needed for an email state change notification
type EmailNotification struct {
	Event     string `json:"event"`
	MessageID string `json:"messageID"`
	Timestamp string `json:"timestamp"`
}

type Webhook struct {
	Event  string `json:"event"`
	Action string `json:"action"`
	Email  Email
}

type Email struct {
	ID string `json:"id"` // message id
}
