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
