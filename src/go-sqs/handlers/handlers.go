package handlers

import (
	"fmt"
	"go-sqs/util"
	"net/url"
	"regexp"
	"strconv"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
)

// CreateQueue TODO: add comment
func CreateQueue(Parameters url.Values, Queues *sync.Map) (string, int) {
	var QueueName = Parameters.Get("QueueName")
	var IsValidQueueName, err = regexp.MatchString("^[a-zA-Z0-9_\\-]{1,80}$", QueueName)
	fmt.Printf("%v", IsValidQueueName)
	fmt.Printf("%v", err)
	if !IsValidQueueName || err != nil {
		return util.Error("InvalidParameterValue", "The specified queue name is not valid.")
	}
	var _, ok = Queues.Load(QueueName)
	if !ok {
		util.CreateQueue(Queues, QueueName)
	}
	var Result = fmt.Sprintf("<QueueUrl>%s</QueueUrl>", QueueName)
	return util.Success("CreateQueue", Result)
}

// DeleteMessage TODO: add comment
func DeleteMessage(Parameters url.Values, Queues *sync.Map) (string, int) {
	var QueueURL = Parameters.Get("QueueUrl")
	var QueuePtr, ok = Queues.Load(QueueURL)
	if !ok {
		return util.Error("AWS.SimpleQueueService.NonExistentQueue", "The specified queue does not exist for this wsdl version.")
	}
	var Queue = QueuePtr.(*util.Queue)
	var ReceiptHandle = Parameters.Get("ReceiptHandle")
	if MessagePtr, ok := Queue.ReceiptHandles.Load(ReceiptHandle); ok {
		var Message = MessagePtr.(*util.Message)
		Queue.ReceiptHandles.Delete(ReceiptHandle)
		Queue.Messages.Delete(Message.MessageID)
	} else {
		return util.Error("ReceiptHandleIsInvalid", fmt.Sprintf("The input receipt handle \"%s\" is not a valid receipt handle.", ReceiptHandle))
	}

	return util.Success("DeleteMessage", "")
}

// DeleteQueue TODO: add comment
func DeleteQueue(Parameters url.Values, Queues *sync.Map) (string, int) {
	var QueueURL = Parameters.Get("QueueUrl")
	var _, ok = Queues.Load(QueueURL)
	if !ok {
		return util.Error("AWS.SimpleQueueService.NonExistentQueue", "The specified queue does not exist for this wsdl version.")
	}
	Queues.Delete(QueueURL)

	return util.Success("DeleteQueue", "")
}

// GetQueueAttributes TODO: add comment
func GetQueueAttributes(Parameters url.Values, Queues *sync.Map) (string, int) {
	var QueueURL = Parameters.Get("QueueUrl")
	var QueuePtr, ok = Queues.Load(QueueURL)
	if !ok {
		return util.Error("AWS.SimpleQueueService.NonExistentQueue", "The specified queue does not exist for this wsdl version.")
	}
	var Queue = QueuePtr.(*util.Queue)

	var NumberOfMessages = 0
	Queue.Messages.Range(func(k, v interface{}) bool {
		NumberOfMessages++
		return true
	})

	var Result = fmt.Sprintf(`<Attribute>
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
  </Attribute>`,
		Queue.QueueArn,
		NumberOfMessages,
		Queue.ApproximateNumberOfMessagesNotVisible,
		Queue.ApproximateNumberOfMessagesDelayed,
		Queue.CreatedTimestamp,
		Queue.LastModifiedTimestamp,
		Queue.VisibilityTimeout,
		Queue.MaximumMessageSize,
		Queue.MessageRetentionPeriod,
		Queue.DelaySeconds,
		Queue.ReceiveMessageWaitTimeSeconds)

	return util.Success("GetQueueAttributes", Result)
}

// GetQueueURL TODO: add comment
func GetQueueURL(Parameters url.Values, Queues *sync.Map) (string, int) {
	var QueueName = Parameters.Get("QueueName")
	var _, ok = Queues.Load(QueueName)
	if !ok {
		return util.Error("AWS.SimpleQueueService.NonExistentQueue", "The specified queue does not exist for this wsdl version.")
	}

	var GetQueueURLResult = fmt.Sprintf("<QueueUrl>%s</QueueUrl>", QueueName)
	return util.Success("GetQueueUrl", GetQueueURLResult)
}

// ListQueues TODO: add comment
func ListQueues(Parameters url.Values, Queues *sync.Map) (string, int) {
	var Result = ""
	Queues.Range(func(QueueURL, v interface{}) bool {
		Result += fmt.Sprintf("<QueueUrl>%s</QueueUrl>", QueueURL)
		return true
	})
	return util.Success("ListQueues", Result)
}

// ReceiveMessage TODO: add comment
func ReceiveMessage(Parameters url.Values, Queues *sync.Map) (string, int) {
	var QueueURL = Parameters.Get("QueueUrl")
	var QueuePtr, ok = Queues.Load(QueueURL)
	if !ok {
		return util.Error("AWS.SimpleQueueService.NonExistentQueue", "The specified queue does not exist for this wsdl version.")
	}
	var Queue = QueuePtr.(*util.Queue)
	var Now = time.Now().Unix()
	var RawVisibilityTimeout = Parameters.Get("VisibilityTimeout")
	var VisibilityTimeout int64 = 30
	if RawVisibilityTimeout != "" {
		var err error
		VisibilityTimeout, err = strconv.ParseInt(RawVisibilityTimeout, 10, 64)
		if err != nil {
			return util.Error("InvalidParameterValue", fmt.Sprintf("Value %s for parameter VisibilityTimeout is invalid. Reason: Must be between 0 and 43200, if provided", RawVisibilityTimeout))
		}
	}

	var FoundMessage *util.Message
	Queue.Messages.Range(func(MessageID, MessagePtr interface{}) bool {
		var Message = MessagePtr.(*util.Message)
		if Message.VisibilityDeadline < Now {
			FoundMessage = Message
			return false
		}
		return true
	})

	var Result = ""

	if FoundMessage != nil {
		FoundMessage.VisibilityDeadline = Now + VisibilityTimeout
		if FoundMessage.ApproximateFirstReceiveTimestamp == 0 {
			FoundMessage.ApproximateFirstReceiveTimestamp = time.Now().Unix()
		}
		FoundMessage.ApproximateReceiveCount++
		if FoundMessage.ReceiptHandle != "" {
			Queue.ReceiptHandles.Delete(FoundMessage.ReceiptHandle)
		}
		FoundMessage.ReceiptHandle = uuid.Must(uuid.NewV4()).String()
		Queue.ReceiptHandles.Store(FoundMessage.ReceiptHandle, FoundMessage)

		Result = fmt.Sprintf(`<Message>
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
    </Message>`, FoundMessage.MessageID, FoundMessage.ReceiptHandle, FoundMessage.Body, FoundMessage.SenderID, FoundMessage.SentTimestamp, FoundMessage.ApproximateReceiveCount, FoundMessage.ApproximateFirstReceiveTimestamp)
	}

	return util.Success("ReceiveMessage", Result)
}

// SendMessage TODO: add comment
func SendMessage(Parameters url.Values, Queues *sync.Map) (string, int) {
	var QueueURL = Parameters.Get("QueueUrl")
	var QueuePtr, ok = Queues.Load(QueueURL)
	if !ok {
		QueuePtr = util.CreateQueue(Queues, QueueURL)
	}
	var Queue = QueuePtr.(*util.Queue)
	var MD5OfMessageBody = ""
	var MD5OfMessageAttributes = ""
	var Message = util.Message{
		MessageID: uuid.Must(uuid.NewV4()).String(),
		Body:      Parameters.Get("MessageBody"),
		SenderID:  "",
		ApproximateFirstReceiveTimestamp: 0,
		ApproximateReceiveCount:          0,
		SentTimestamp:                    time.Now().Unix(),
		VisibilityDeadline:               0,
	}
	Queue.Messages.Store(Message.MessageID, &Message)

	var Result = fmt.Sprintf(`<MD5OfMessageBody>%s</MD5OfMessageBody>
	<MD5OfMessageAttributes>%s</MD5OfMessageAttributes>
	<MessageId>%s</MessageId>`, MD5OfMessageBody, MD5OfMessageAttributes, Message.MessageID)

	return util.Success("SendMessage", Result)
}
