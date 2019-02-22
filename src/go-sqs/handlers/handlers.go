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

type batchValidationResult struct {
	Ok           bool
	BatchSize    int
	ErrorCode    string
	ErrorMessage string
}

// CreateQueue TODO: add comment
func CreateQueue(Parameters url.Values, Queues *sync.Map) (string, int) {
	var QueueName = Parameters.Get("QueueName")
	var IsValidQueueName, err = regexp.MatchString("^[a-zA-Z0-9_\\-]{1,80}$", QueueName)
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

func validateBatch(Parameters url.Values, BatchPrefix string, RequiredKeys []string) batchValidationResult {
	// TODO: Validate if there are dangling IDs / IDs starting not from 1 / etc
	// TODO: Validate if there are incomplete pairs of ReceiptHandle/Id entries
	// TODO: Validate if batch request is not empty (AWS.SimpleQueueService.EmptyBatchRequest)
	// TODO: Validate that there are no more than 10 entries in batch
	// TODO: Validate that IDs are distinct (AWS.SimpleQueueService.BatchEntryIdsNotDistinct)
	// TODO: Validate that all required RequiredKeys are present
	var BatchValidationResult = batchValidationResult{
		Ok: false,
	}
	var BatchSize = 0

	for i := 1; ; i++ {
		BatchEntryIDs, ok := Parameters[fmt.Sprintf("%s.%d.Id", BatchPrefix, i)]
		if !ok {
			break
		}

		if i == 11 {
			BatchValidationResult.ErrorCode = "AWS.SimpleQueueService.TooManyEntriesInBatchRequest"
			BatchValidationResult.ErrorMessage = "The batch request contains more entries than permissible."
			return BatchValidationResult
		}

		if len(BatchEntryIDs) != 1 {
			BatchValidationResult.ErrorCode = "InvalidQueryParameter"
			BatchValidationResult.ErrorMessage = "The AWS query string is malformed or does not adhere to AWS standards."
			return BatchValidationResult
		}

		var BatchEntryID = BatchEntryIDs[0]

		var IsValidBatchEntryID, err = regexp.MatchString("^[a-zA-Z0-9_\\-]{1,80}$", BatchEntryID)
		if !IsValidBatchEntryID || err != nil {
			BatchValidationResult.ErrorCode = "AWS.SimpleQueueService.InvalidBatchEntryId"
			BatchValidationResult.ErrorMessage = "The Id of a batch entry in a batch request doesn't abide by the specification."
			return BatchValidationResult
		}

		BatchSize++
	}

	if BatchSize == 0 {
		BatchValidationResult.ErrorCode = "AWS.SimpleQueueService.EmptyBatchRequest"
		BatchValidationResult.ErrorMessage = "The batch request doesn't contain any entries."
		return BatchValidationResult
	}

	BatchValidationResult.BatchSize = BatchSize
	BatchValidationResult.Ok = true
	return BatchValidationResult
}

func deleteMessage(Queue *util.Queue, ReceiptHandle string) bool {
	if MessagePtr, ok := Queue.ReceiptHandles.Load(ReceiptHandle); ok {
		var Message = MessagePtr.(*util.Message)
		Queue.ReceiptHandles.Delete(ReceiptHandle)
		// Queue.Messages.Delete(Message.MessageID)
		delete(Queue.Messages2, Message.MessageID)
		return true
	}

	return false
}

// DeleteMessage TODO: add comment
func DeleteMessage(Parameters url.Values, Queues *sync.Map) (string, int) {
	var QueueURL = Parameters.Get("QueueUrl")
	var QueuePtr, ok = Queues.Load(QueueURL)
	if !ok {
		return util.Error("AWS.SimpleQueueService.NonExistentQueue", "The specified queue does not exist for this wsdl version.")
	}
	var Queue = QueuePtr.(*util.Queue)

	// TODO: Validate ReceiptHandle format
	var ReceiptHandle = Parameters.Get("ReceiptHandle")
	ok = deleteMessage(Queue, ReceiptHandle)
	if !ok {
		return util.Error("ReceiptHandleIsInvalid", fmt.Sprintf("The input receipt handle \"%s\" is not a valid receipt handle.", ReceiptHandle))
	}

	return util.Success("DeleteMessage", "")
}

// DeleteMessageBatch TODO: add comment
func DeleteMessageBatch(Parameters url.Values, Queues *sync.Map) (string, int) {
	var QueueURL = Parameters.Get("QueueUrl")
	var QueuePtr, ok = Queues.Load(QueueURL)
	if !ok {
		return util.Error("AWS.SimpleQueueService.NonExistentQueue", "The specified queue does not exist for this wsdl version.")
	}
	var Queue = QueuePtr.(*util.Queue)

	var BatchValidationResult = validateBatch(Parameters, "DeleteMessageBatchRequestEntry", []string{
		"Id",
		"ReceiptHandle",
	}[:])
	if !BatchValidationResult.Ok {
		return util.Error(BatchValidationResult.ErrorCode, BatchValidationResult.ErrorMessage)
	}

	var SuccessfulResult = ""
	var ErrorResult = ""
	for i := 1; i <= BatchValidationResult.BatchSize; i++ {
		var ReceiptHandleID = Parameters.Get(fmt.Sprintf("DeleteMessageBatchRequestEntry.%d.Id", i))
		// TODO: Validate ReceiptHandle format
		var ReceiptHandle = Parameters.Get(fmt.Sprintf("DeleteMessageBatchRequestEntry.%d.ReceiptHandle", i))

		ok = deleteMessage(Queue, ReceiptHandle)
		if ok {
			SuccessfulResult += fmt.Sprintf("<DeleteMessageBatchResultEntry><Id>%s</Id></DeleteMessageBatchResultEntry>", ReceiptHandleID)
		} else {
			ErrorResult += fmt.Sprintf("<BatchResultErrorEntry><Id>%s</Id><Code>ReceiptHandleIsInvalid</Code><Message>The input receipt handle is invalid.</Message><SenderFault>true</SenderFault></BatchResultErrorEntry>", ReceiptHandleID)
		}
	}

	var Result = SuccessfulResult + ErrorResult

	return util.Success("DeleteMessageBatch", Result)
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

	// var NumberOfMessages = 0
	// Queue.Messages.Range(func(k, v interface{}) bool {
	// NumberOfMessages++
	// return true
	// })
	var NumberOfMessages = len(Queue.Messages2)

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
	var VisibilityTimeout = 30
	// TODO: Validate VisibilityTimeout value properly
	if RawVisibilityTimeout != "" {
		var err error
		VisibilityTimeout, err = strconv.Atoi(RawVisibilityTimeout)
		if err != nil {
			return util.Error("InvalidParameterValue", fmt.Sprintf("Value %s for parameter VisibilityTimeout is invalid. Reason: Must be between 0 and 43200, if provided", RawVisibilityTimeout))
		}
	}

	var RawMaxNumberOfMessages = Parameters.Get("MaxNumberOfMessages")
	var MaxNumberOfMessages = 1
	if RawMaxNumberOfMessages != "" {
		var err error
		MaxNumberOfMessages, err = strconv.Atoi(RawMaxNumberOfMessages)
		if err != nil || MaxNumberOfMessages < 1 || MaxNumberOfMessages > 10 {
			return util.Error("InvalidParameterValue", fmt.Sprintf("Value %s for parameter MaxNumberOfMessages is invalid. Reason: Must be between 1 and 10", RawMaxNumberOfMessages))
		}
	}

	var FoundMessages [10]*util.Message
	var NumFoundMessages = 0
	for _, Message := range Queue.Messages2 {
		if Message.VisibilityDeadline < Now {
			FoundMessages[NumFoundMessages] = Message
			NumFoundMessages++

			if NumFoundMessages == MaxNumberOfMessages {
				break
			}
		}
	}
	// Queue.Messages.Range(func(MessageID, MessagePtr interface{}) bool {
	// 	var Message = MessagePtr.(*util.Message)
	// 	if Message.VisibilityDeadline < Now {
	// 		FoundMessages[NumFoundMessages] = Message
	// 		NumFoundMessages++

	// 		if NumFoundMessages == MaxNumberOfMessages {
	// 			return false
	// 		}
	// 	}
	// 	return true
	// })

	var Result = ""

	if NumFoundMessages > 0 {
		for i := 0; i < NumFoundMessages; i++ {
			var FoundMessage = FoundMessages[i]
			FoundMessage.VisibilityDeadline = Now + int64(VisibilityTimeout)
			if FoundMessage.ApproximateFirstReceiveTimestamp == 0 {
				FoundMessage.ApproximateFirstReceiveTimestamp = time.Now().Unix()
			}
			FoundMessage.ApproximateReceiveCount++
			if FoundMessage.ReceiptHandle != "" {
				Queue.ReceiptHandles.Delete(FoundMessage.ReceiptHandle)
			}
			FoundMessage.ReceiptHandle = uuid.Must(uuid.NewV4()).String()
			Queue.ReceiptHandles.Store(FoundMessage.ReceiptHandle, FoundMessage)

			Result += fmt.Sprintf(`<Message>
			<MessageId>%s</MessageId>
			<ReceiptHandle>%s</ReceiptHandle>
			<MD5OfBody>%s</MD5OfBody>
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
			</Message>`, FoundMessage.MessageID, FoundMessage.ReceiptHandle, FoundMessage.MD5OfMessageBody, FoundMessage.Body, FoundMessage.SenderID, FoundMessage.SentTimestamp, FoundMessage.ApproximateReceiveCount, FoundMessage.ApproximateFirstReceiveTimestamp)
		}
	}

	return util.Success("ReceiveMessage", Result)
}

func sendMessage(Queue *util.Queue, MessageBody string, DelaySeconds int) (*util.Message, bool, string, string) {
	// TODO: Move validation of delay seconds to this method
	// TODO: Calculate MD5 of message body
	var MD5OfMessageBody = ""
	// TODO: Calculate MD5 of message attributes
	var MD5OfMessageAttributes = ""

	if DelaySeconds < 0 || DelaySeconds > 900 {
		return nil, false, "InvalidParameterValue", fmt.Sprintf("Value %d for parameter DelaySeconds is invalid. Reason: Must be between 0 and 900, if provided.", DelaySeconds)
	}

	var VisibilityDeadline int64
	if DelaySeconds > 0 {
		VisibilityDeadline = time.Now().Unix() + int64(DelaySeconds)
	}

	var Message = util.Message{
		MessageID:              uuid.Must(uuid.NewV4()).String(),
		MD5OfMessageBody:       MD5OfMessageBody,
		MD5OfMessageAttributes: MD5OfMessageAttributes,
		Body:     MessageBody,
		SenderID: "",
		ApproximateFirstReceiveTimestamp: 0,
		ApproximateReceiveCount:          0,
		SentTimestamp:                    time.Now().Unix(),
		VisibilityDeadline:               VisibilityDeadline,
	}

	Queue.SendChannel <- &Message
	return &Message, true, "", ""
}

// SendMessage TODO: add comment
func SendMessage(Parameters url.Values, Queues *sync.Map) (string, int) {
	var QueueURL = Parameters.Get("QueueUrl")
	var QueuePtr, ok = Queues.Load(QueueURL)
	if !ok {
		// TODO: Disable automatic queue creation on send
		QueuePtr = util.CreateQueue(Queues, QueueURL)
	}
	var Queue = QueuePtr.(*util.Queue)

	var RawDelaySeconds = Parameters.Get("DelaySeconds")
	var DelaySeconds = Queue.DelaySeconds
	if RawDelaySeconds != "" {
		var err error
		if DelaySeconds, err = strconv.Atoi(RawDelaySeconds); err != nil {
			return util.Error("InvalidParameterValue", fmt.Sprintf("Parameter DelaySeconds should be of type Integer"))
		}
	}

	var Message *util.Message
	var ErrorCode, ErrorMessage string
	Message, ok, ErrorCode, ErrorMessage = sendMessage(Queue, Parameters.Get("MessageBody"), DelaySeconds)

	if !ok {
		return util.Error(ErrorCode, ErrorMessage)
	}
	var Result = fmt.Sprintf(`<MD5OfMessageBody>%s</MD5OfMessageBody>
	<MD5OfMessageAttributes>%s</MD5OfMessageAttributes>
	<MessageId>%s</MessageId>`, Message.MD5OfMessageBody, Message.MD5OfMessageAttributes, Message.MessageID)

	return util.Success("SendMessage", Result)
}

// SendMessageBatch TODO: add comment
func SendMessageBatch(Parameters url.Values, Queues *sync.Map) (string, int) {
	var QueueURL = Parameters.Get("QueueUrl")
	var QueuePtr, ok = Queues.Load(QueueURL)
	if !ok {
		return util.Error("AWS.SimpleQueueService.NonExistentQueue", "The specified queue does not exist for this wsdl version.")
	}
	var Queue = QueuePtr.(*util.Queue)

	var BatchValidationResult = validateBatch(Parameters, "SendMessageBatchRequestEntry", []string{
		"Id",
		"MessageBody",
	}[:])
	if !BatchValidationResult.Ok {
		return util.Error(BatchValidationResult.ErrorCode, BatchValidationResult.ErrorMessage)
	}

	var SuccessfulResult = ""
	var ErrorResult = ""
	for i := 1; i <= BatchValidationResult.BatchSize; i++ {
		var BatchEntryID = Parameters.Get(fmt.Sprintf("SendMessageBatchRequestEntry.%d.Id", i))
		var MessageBody = Parameters.Get(fmt.Sprintf("SendMessageBatchRequestEntry.%d.MessageBody", i))

		var DelaySeconds = Queue.DelaySeconds
		var RawDelaySeconds = Parameters.Get(fmt.Sprintf("SendMessageBatchRequestEntry.%d.DelaySeconds", i))
		if RawDelaySeconds != "" {
			var err error
			DelaySeconds, err = strconv.Atoi(RawDelaySeconds)
			if err != nil {
				ErrorResult += fmt.Sprintf("<BatchResultErrorEntry><Id>%s</Id><Code>InvalidParameterValue</Code><Message>Parameter DelaySeconds should be of type Integer</Message><SenderFault>true</SenderFault></BatchResultErrorEntry>", BatchEntryID)
				continue
			}
		}

		var Message, ok, ErrorCode, ErrorMessage = sendMessage(Queue, MessageBody, DelaySeconds)
		if ok {
			SuccessfulResult += fmt.Sprintf("<SendMessageBatchResultEntry><Id>%s</Id><MD5OfMessageAttributes>%s</MD5OfMessageAttributes><MD5OfMessageBody>%s</MD5OfMessageBody><MessageId>%s</MessageId></SendMessageBatchResultEntry>", BatchEntryID, Message.MD5OfMessageBody, Message.MD5OfMessageAttributes, Message.MessageID)
		} else {
			ErrorResult += fmt.Sprintf("<BatchResultErrorEntry><Id>%s</Id><Code>%s</Code><Message>%s</Message><SenderFault>true</SenderFault></BatchResultErrorEntry>", BatchEntryID, ErrorCode, ErrorMessage)
		}
	}

	var Result = SuccessfulResult + ErrorResult

	return util.Success("SendMessageBatch", Result)
}
