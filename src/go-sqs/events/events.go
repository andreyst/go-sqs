package events

// ReceiveRequestEvent represent a request to receive a message
type ReceiveRequestEvent struct {
	MaxNumberOfMessages int
	VisibilityTimeout   int
	ReturnChan          chan ReceiveResponseEvent
}

// ReceiveResponseEvent represent a response to ReceiveEvent.
type ReceiveResponseEvent struct {
	// TODO: Break dependency cycle and make Message a concrete type
	Messages []interface{}
}

// DeleteRequestEvent represents a request to delete a message.
type DeleteRequestEvent struct {
	ReceiptHandle string
	ReturnChan    chan DeleteResponseEvent
}

// DeleteResponseEvent represents a response to a request to delete a message.
type DeleteResponseEvent struct {
	Ok bool
}
