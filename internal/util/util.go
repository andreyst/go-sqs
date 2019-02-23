package util

import (
	"fmt"
	"sync"
	"time"

	"github.com/andreyst/go-sqs/internal/events"
	"github.com/andreyst/go-sqs/internal/limits"
	"github.com/andreyst/go-sqs/internal/queue"
	uuid "github.com/satori/go.uuid"
)

// GetQueueByName TODO: Add comment
func GetQueueByName(Queues *sync.Map, QueueName string) (*queue.Queue, string) {
	// TODO: Refactor to use map instead of full scan
	var FoundQueue *queue.Queue
	var FoundQueueURL string
	Queues.Range(func(QueueURL, v interface{}) bool {
		var Queue = v.(*queue.Queue)
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
func CreateQueue(Queues *sync.Map, QueueName string) (*queue.Queue, string) {
	var ExistingQueue, ExistingQueueURL = GetQueueByName(Queues, QueueName)
	if ExistingQueue != nil {
		return ExistingQueue, ExistingQueueURL
	}

	// TODO: Move protocol/hostname prefix to app config
	var QueueURL = fmt.Sprintf("http://localhost/%s", QueueName)
	var Queue = queue.Queue{
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
		Messages2:                             make(map[string]*queue.Message),
		SendChannel:                           make(chan *queue.Message),
		ReceiveChannel:                        make(chan events.ReceiveRequestEvent),
		DeleteChannel:                         make(chan events.DeleteRequestEvent),
	}
	Queues.Store(QueueURL, &Queue)

	go queueActor(&Queue)
	return &Queue, QueueURL
}

func queueActor(Queue *queue.Queue) {
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

func sendMessage(Queue *queue.Queue, Message *queue.Message) {
	// TODO: Check if there's a race condition with reading this field somewhere else
	Queue.ApproximateNumberOfMessages++
	Queue.Messages2[Message.MessageID] = Message
}

func receiveMessage(Queue *queue.Queue, Event events.ReceiveRequestEvent) {
	var Now = time.Now().Unix()
	var FoundMessages = make([]interface{}, limits.MaxBatchSize)
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

func deleteMessage(Queue *queue.Queue, Event events.DeleteRequestEvent) {
	var MessagePtr, ok = Queue.ReceiptHandles.Load(Event.ReceiptHandle)
	if !ok {
		Event.ReturnChan <- events.DeleteResponseEvent{
			Ok: false,
		}
		return
	}

	var Message = MessagePtr.(*queue.Message)
	Queue.ReceiptHandles.Delete(Event.ReceiptHandle)
	delete(Queue.Messages2, Message.MessageID)
	Queue.ApproximateNumberOfMessages--
	Event.ReturnChan <- events.DeleteResponseEvent{
		Ok: true,
	}
}
