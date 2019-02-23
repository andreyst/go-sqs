package queue

import (
	"sync"

	"github.com/andreyst/go-sqs/internal/events"
)

// Queue TODO: add comment
type Queue struct {
	QueueName                             string
	QueueArn                              string
	ApproximateNumberOfMessages           int64
	ApproximateNumberOfMessagesNotVisible int64
	ApproximateNumberOfMessagesDelayed    int64
	CreatedTimestamp                      int64
	LastModifiedTimestamp                 int64
	VisibilityTimeout                     int
	MaximumMessageSize                    int
	MessageRetentionPeriod                int
	DelaySeconds                          int
	ReceiveMessageWaitTimeSeconds         int
	Messages                              sync.Map
	Messages2                             map[string]*Message
	SendChannel                           chan *Message
	ReceiveChannel                        chan events.ReceiveRequestEvent
	DeleteChannel                         chan events.DeleteRequestEvent
	ReceiptHandles                        sync.Map
}
