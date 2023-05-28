package env

import "os"

var (
	// AWS Region
	Region = os.Getenv("REGION")

	TableName            = os.Getenv("DYNAMODB_TABLE")
	GsiOriginalIndexName = os.Getenv("DYNAMODB_ORIGINAL_INDEX")
	GsiIndexName         = os.Getenv("DYNAMODB_TIME_INDEX")
	S3Bucket             = os.Getenv("S3_BUCKET")
	QueueName            = os.Getenv("SQS_QUEUE")

	WebhookURL = os.Getenv("WEBHOOK_URL")
)
