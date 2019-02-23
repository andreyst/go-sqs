package queue

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
