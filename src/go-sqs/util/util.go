package util

import (
	"fmt"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
)

// Queue TODO: add comment
type Queue struct {
	QueueArn                              string
	ApproximateNumberOfMessages           int64
	ApproximateNumberOfMessagesNotVisible int64
	ApproximateNumberOfMessagesDelayed    int64
	CreatedTimestamp                      int64
	LastModifiedTimestamp                 int64
	VisibilityTimeout                     int64
	MaximumMessageSize                    int64
	MessageRetentionPeriod                int64
	DelaySeconds                          int64
	ReceiveMessageWaitTimeSeconds         int64
	Messages                              sync.Map
	ReceiptHandles                        sync.Map
}

// Message TODO: add comment
type Message struct {
	MessageID                        string
	Body                             string
	SenderID                         string
	ReceiptHandle                    string
	ApproximateFirstReceiveTimestamp int64
	ApproximateReceiveCount          int64
	SentTimestamp                    int64
	VisibilityDeadline               int64
}

// CreateRequestID TODO: add comment
func CreateRequestID() string {
	return uuid.Must(uuid.NewV4()).String()
}

// Success TODO: add comment
func Success(Action string, Result string) (string, int) {
	var RequestID = CreateRequestID()
	return fmt.Sprintf(`<%sResponse>
	<%sResult>%s</%sResult>
	<ResponseMetadata>
		<RequestId>%s</RequestId>
	</ResponseMetadata>
</%sResponse>`, Action, Action, Result, Action, RequestID, Action), 200
}

// Error TODO: add comment
func Error(ErrorCode string, ErrorMessage string) (string, int) {
	var RequestID = CreateRequestID()
	return fmt.Sprintf(`<ErrorResponse>
  <Error>
    <Type>Sender</Type>
    <Code>%s</Code>
    <Message>%s</Message>
    <Detail/>
  </Error>
  <RequestId>%s</RequestId>
</ErrorResponse>`, ErrorCode, ErrorMessage, RequestID), 400
}

// CreateQueue TODO: Add comment
func CreateQueue(Queues *sync.Map, QueueName string) *Queue {
	// TODO: Create real URL for queue
	Queues.Store(QueueName, &Queue{
		// TODO: Create good ARN
		QueueArn:                              QueueName,
		ApproximateNumberOfMessages:           0,
		ApproximateNumberOfMessagesNotVisible: 0,
		ApproximateNumberOfMessagesDelayed:    0,
		CreatedTimestamp:                      time.Now().Unix(),
		LastModifiedTimestamp:                 time.Now().Unix(),
		VisibilityTimeout:                     0,
		MaximumMessageSize:                    262144,
		MessageRetentionPeriod:                346500,
		DelaySeconds:                          0,
		ReceiveMessageWaitTimeSeconds:         30,
	})

	var QueuePtr, _ = Queues.Load(QueueName)
	return QueuePtr.(*Queue)
}
