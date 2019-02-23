package util

import (
	"fmt"
	"sync"
	"time"

	"github.com/andreyst/go-sqs/internal/events"
	uuid "github.com/satori/go.uuid"
)

// MaxBatchSize defines what could be the maximum size of the batch in SQS.
// It drives constraints in receive functions and in validators.
const MaxBatchSize = 10

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

// Message TODO: add comment
type Message struct {
	MessageID                        string
	Body                             string
	MD5OfMessageBody                 string
	MD5OfMessageAttributes           string
	SenderID                         string
	ReceiptHandle                    string
	ApproximateFirstReceiveTimestamp int64
	ApproximateReceiveCount          int
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

// GetQueueByName TODO: Add comment
func GetQueueByName(Queues *sync.Map, QueueName string) (*Queue, string) {
	// TODO: Refactor to use map instead of full scan
	var FoundQueue *Queue
	var FoundQueueURL string
	Queues.Range(func(QueueURL, v interface{}) bool {
		var Queue = v.(*Queue)
		if Queue.QueueName == QueueName {
			FoundQueue = Queue
			FoundQueueURL = QueueURL.(string)
			return false
		}
		return true
	})

	return FoundQueue, FoundQueueURL
}

// CreateQueue TODO: Add comment
func CreateQueue(Queues *sync.Map, QueueName string) (*Queue, string) {
	var ExistingQueue, ExistingQueueURL = GetQueueByName(Queues, QueueName)
	if ExistingQueue != nil {
		return ExistingQueue, ExistingQueueURL
	}

	// TODO: Move protocol/hostname prefix to app config
	var QueueURL = fmt.Sprintf("http://localhost/%s", QueueName)
	var Queue = Queue{
		QueueName: QueueName,
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
		Messages2:                             make(map[string]*Message),
		SendChannel:                           make(chan *Message),
		ReceiveChannel:                        make(chan events.ReceiveRequestEvent),
		DeleteChannel:                         make(chan events.DeleteRequestEvent),
	}
	Queues.Store(QueueURL, &Queue)

	go queueActor(&Queue)
	return &Queue, QueueURL
}

func queueActor(Queue *Queue) {
	for {
		select {
		case Message := <-Queue.SendChannel:
			sendMessage(Queue, Message)
		case event := <-Queue.ReceiveChannel:
			receiveMessage(Queue, event)
		case event := <-Queue.DeleteChannel:
			deleteMessage(Queue, event)
		}
	}
}

func sendMessage(Queue *Queue, Message *Message) {
	// TODO: Check if there's a race condition with reading this field somewhere else
	Queue.ApproximateNumberOfMessages++
	Queue.Messages2[Message.MessageID] = Message
}

func receiveMessage(Queue *Queue, Event events.ReceiveRequestEvent) {
	var Now = time.Now().Unix()
	var FoundMessages = make([]interface{}, MaxBatchSize)
	var NumFoundMessages = 0
	for _, Message := range Queue.Messages2 {
		if Message.VisibilityDeadline < Now {
			FoundMessages[NumFoundMessages] = Message

			if Message.ReceiptHandle != "" {
				Queue.ReceiptHandles.Delete(Message.ReceiptHandle)
			}
			// TODO: Refactor to put both message ID and unique receipt handle inside
			// eliminate the need for separate map for receipt handles
			Message.ReceiptHandle = uuid.Must(uuid.NewV4()).String()
			Queue.ReceiptHandles.Store(Message.ReceiptHandle, Message)
			Message.VisibilityDeadline = Now + int64(Event.VisibilityTimeout)
			if Message.ApproximateFirstReceiveTimestamp == 0 {
				Message.ApproximateFirstReceiveTimestamp = time.Now().Unix()
			}
			Message.ApproximateReceiveCount++

			NumFoundMessages++
			if NumFoundMessages == Event.MaxNumberOfMessages {
				break
			}
		}
	}

	Event.ReturnChan <- events.ReceiveResponseEvent{
		Messages: FoundMessages[:NumFoundMessages],
	}
}

func deleteMessage(Queue *Queue, Event events.DeleteRequestEvent) {
	var MessagePtr, ok = Queue.ReceiptHandles.Load(Event.ReceiptHandle)
	if !ok {
		Event.ReturnChan <- events.DeleteResponseEvent{
			Ok: false,
		}
		return
	}

	var Message = MessagePtr.(*Message)
	Queue.ReceiptHandles.Delete(Event.ReceiptHandle)
	delete(Queue.Messages2, Message.MessageID)
	Queue.ApproximateNumberOfMessages--
	Event.ReturnChan <- events.DeleteResponseEvent{
		Ok: true,
	}
}
