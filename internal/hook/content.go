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

type Hook struct {
	Event     string `json:"event"`
	Action    string `json:"action"`
	Timestamp string `json:"timestamp"`
	Email     Email
}

type Email struct {
	ID string `json:"id"` // message id
}
