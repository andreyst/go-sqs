package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
  "sync"

	uuid "github.com/satori/go.uuid"
)

// Queue - SQS queue.
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
}

// Message - SQS message.
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

// Queues - queues map.
var Queues sync.Map

// Messages - messages map.
var Messages sync.Map

// ReceiptHandles - receipt handles map.
var ReceiptHandles sync.Map

func _createQueue(QueueName string) {
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
}

func listQueues(w http.ResponseWriter, r *http.Request) {
	var RequestID = uuid.Must(uuid.NewV4()).String()
	var ListQueuesResult = ""
  Queues.Range(func(QueueURL, v interface{}) bool {
    ListQueuesResult += fmt.Sprintf("<QueueUrl>%s</QueueUrl>", QueueURL)
    return true
  })
	fmt.Fprintf(w, `
<ListQueuesResponse>
    <ListQueuesResult>
        `+ListQueuesResult+`
    </ListQueuesResult>
    <ResponseMetadata>
        <RequestId>`+RequestID+`</RequestId>
    </ResponseMetadata>
</ListQueuesResponse>
`)
}

func getQueueAttributes(w http.ResponseWriter, r *http.Request) {
	var QueueURL = r.Form.Get("QueueUrl")
	var QueuePtr, ok = Queues.Load(QueueURL)
	if !ok {
		writeError(w, r, "AWS.SimpleQueueService.NonExistentQueue", "The specified queue does not exist for this wsdl version.")
		return
	}
  var Queue = QueuePtr.(Queue)

	var RequestID = uuid.Must(uuid.NewV4()).String()
	fmt.Fprintf(w, `<GetQueueAttributesResponse>
  <GetQueueAttributesResult>
    <Attribute>
      <Name>QueueArn</Name>
      <Value>%s</Value>
    </Attribute>
    <Attribute>
      <Name>ApproximateNumberOfMessages</Name>
      <Value>%d</Value>
    </Attribute>
    <Attribute>
      <Name>ApproximateNumberOfMessagesNotVisible</Name>
      <Value>%d</Value>
    </Attribute>
    <Attribute>
      <Name>ApproximateNumberOfMessagesDelayed</Name>
      <Value>%d</Value>
    </Attribute>
    <Attribute>
      <Name>CreatedTimestamp</Name>
      <Value>%d</Value>
    </Attribute>
    <Attribute>
      <Name>LastModifiedTimestamp</Name>
      <Value>%d</Value>
    </Attribute>
    <Attribute>
      <Name>VisibilityTimeout</Name>
      <Value>%d</Value>
    </Attribute>
    <Attribute>
      <Name>MaximumMessageSize</Name>
      <Value>%d</Value>
    </Attribute>
    <Attribute>
      <Name>MessageRetentionPeriod</Name>
      <Value>%d</Value>
    </Attribute>
    <Attribute>
      <Name>DelaySeconds</Name>
      <Value>%d</Value>
    </Attribute>
    <Attribute>
      <Name>ReceiveMessageWaitTimeSeconds</Name>
      <Value>%d</Value>
    </Attribute>
  </GetQueueAttributesResult>
  <ResponseMetadata>
    <RequestId>`+RequestID+`</RequestId>
  </ResponseMetadata>
</GetQueueAttributesResponse>`,
		Queue.QueueArn,
    0,
		Queue.ApproximateNumberOfMessagesNotVisible,
		Queue.ApproximateNumberOfMessagesDelayed,
		Queue.CreatedTimestamp,
		Queue.LastModifiedTimestamp,
		Queue.VisibilityTimeout,
		Queue.MaximumMessageSize,
		Queue.MessageRetentionPeriod,
		Queue.DelaySeconds,
		Queue.ReceiveMessageWaitTimeSeconds)
}

func sendMessage(w http.ResponseWriter, r *http.Request) {
	var QueueURL = r.Form.Get("QueueUrl")
	var _, ok = Queues.Load("QueueURL")
	if !ok {
		_createQueue(QueueURL)
	}
	var RequestID = uuid.Must(uuid.NewV4()).String()
	var MD5OfMessageBody = ""
	var MD5OfMessageAttributes = ""
	var Message = Message{
		MessageID: uuid.Must(uuid.NewV4()).String(),
		Body:      r.Form.Get("MessageBody"),
		SenderID:  "",
		ApproximateFirstReceiveTimestamp: 0,
		ApproximateReceiveCount:          0,
		SentTimestamp:                    time.Now().Unix(),
		VisibilityDeadline:               0,
	}
	// Messages[Message.MessageID] = &Message
  Messages.Store(Message.MessageID, &Message)
	fmt.Fprintf(w, `<SendMessageResponse>
    <SendMessageResult>
        <MD5OfMessageBody>`+MD5OfMessageBody+`</MD5OfMessageBody>
        <MD5OfMessageAttributes>`+MD5OfMessageAttributes+`</MD5OfMessageAttributes>
        <MessageId>`+Message.MessageID+`</MessageId>
    </SendMessageResult>
    <ResponseMetadata>
      <RequestId>`+RequestID+`</RequestId>
    </ResponseMetadata>
</SendMessageResponse>`)
}

func receiveMessage(w http.ResponseWriter, r *http.Request) {
	var Now = time.Now().Unix()
	var RawVisibilityTimeout = r.Form.Get("VisibilityTimeout")
	var VisibilityTimeout int64 = 30
	if RawVisibilityTimeout != "" {
		var err error
		VisibilityTimeout, err = strconv.ParseInt(RawVisibilityTimeout, 10, 64)
		if err != nil {
			writeError(w, r, "InvalidParameterValue", fmt.Sprintf("Value %s for parameter VisibilityTimeout is invalid. Reason: Must be between 0 and 43200, if provided", RawVisibilityTimeout))
			return
		}
	}

	var FoundMessage *Message
  Queues.Range(func(MessageID, MessagePtr interface{}) bool {
    var Message = MessagePtr.(*Message)
    if Message.VisibilityDeadline < Now {
      FoundMessage = Message
      return false
    }
    return true
  })

	var ReceiveMessageResult = ""

	if FoundMessage != nil {
		FoundMessage.VisibilityDeadline = Now + VisibilityTimeout
		if FoundMessage.ApproximateFirstReceiveTimestamp == 0 {
			FoundMessage.ApproximateFirstReceiveTimestamp = time.Now().Unix()
		}
		FoundMessage.ApproximateReceiveCount++
		if FoundMessage.ReceiptHandle != "" {
			ReceiptHandles.Delete(FoundMessage.ReceiptHandle)
		}
		FoundMessage.ReceiptHandle = uuid.Must(uuid.NewV4()).String()
		ReceiptHandles.Store(FoundMessage.ReceiptHandle, FoundMessage)

		ReceiveMessageResult = fmt.Sprintf(`
    <Message>
    <MessageId>%s</MessageId>
    <ReceiptHandle>%s</ReceiptHandle>
    <MD5OfBody></MD5OfBody>
    <Body>%s</Body>
    <Attribute>
      <Name>SenderId</Name>
      <Value>%s</Value>
    </Attribute>
    <Attribute>
      <Name>SentTimestamp</Name>
      <Value>%d</Value>
    </Attribute>
    <Attribute>
      <Name>ApproximateReceiveCount</Name>
      <Value>%d</Value>
    </Attribute>
    <Attribute>
      <Name>ApproximateFirstReceiveTimestamp</Name>
      <Value>%d</Value>
    </Attribute>
    </Message>
`, FoundMessage.MessageID, FoundMessage.ReceiptHandle, FoundMessage.Body, FoundMessage.SenderID, FoundMessage.SentTimestamp, FoundMessage.ApproximateReceiveCount, FoundMessage.ApproximateFirstReceiveTimestamp)
	}
	var RequestID = uuid.Must(uuid.NewV4()).String()
	fmt.Fprintf(w, `<ReceiveMessageResponse>
  <ReceiveMessageResult>`+ReceiveMessageResult+`
  </ReceiveMessageResult>
  <ResponseMetadata>
    <RequestId>`+RequestID+`</RequestId>
  </ResponseMetadata>
</ReceiveMessageResponse>`)
}

func deleteMessage(w http.ResponseWriter, r *http.Request) {
	var ReceiptHandle = r.Form.Get("ReceiptHandle")
	if MessagePtr, ok := ReceiptHandles.Load(ReceiptHandle); ok {
    var Message = MessagePtr.(Message)
    ReceiptHandles.Delete(ReceiptHandle)
    Messages.Delete(Message.MessageID)
	} else {
		writeError(w, r, "ReceiptHandleIsInvalid", fmt.Sprintf("The input receipt handle \"%s\" is not a valid receipt handle.", ReceiptHandle))
		return
	}

	var RequestID = uuid.Must(uuid.NewV4()).String()
	fmt.Fprintf(w, `
<DeleteMessageResponse>
    <ResponseMetadata>
        <RequestId>`+RequestID+`</RequestId>
    </ResponseMetadata>
</DeleteMessageResponse>
`)
}

func writeError(w http.ResponseWriter, r *http.Request, ErrorCode string, ErrorMessage string) {
	var RequestID = uuid.Must(uuid.NewV4()).String()
	fmt.Fprintf(w, `
<ErrorResponse>
  <Error>
    <Type>Sender</Type>
    <Code>%s</Code>
    <Message>%s</Message>
    <Detail/>
  </Error>
  <RequestId>%s</RequestId>
</ErrorResponse>
  `, ErrorCode, ErrorMessage, RequestID)
}

func handler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	var act = r.Form.Get("Action")
	switch act {
	case "ListQueues":
		listQueues(w, r)
	case "GetQueueAttributes":
		getQueueAttributes(w, r)
	case "SendMessage":
		sendMessage(w, r)
	case "ReceiveMessage":
		receiveMessage(w, r)
	case "DeleteMessage":
		deleteMessage(w, r)
	default:
		fmt.Printf("Unknown Action: %s\n", act)
	}
}

func main() {
  var Port = "8080"
  if len(os.Args) > 0 {
    Port = os.Args[1]
  }
  http.HandleFunc("/", handler)
  log.Fatal(http.ListenAndServe(":" + Port, nil))
}
