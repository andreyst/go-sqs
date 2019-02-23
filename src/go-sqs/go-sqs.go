package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	handlers "go-sqs/handlers"
	util "go-sqs/util"
)

// Queues TODO: add comment
var queues sync.Map

func handler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	var Action = r.Form.Get("Action")
	var Response = ""
	var StatusCode = 0
	switch Action {
	case "CreateQueue":
		Response, StatusCode = handlers.CreateQueue(r.Form, &queues)
	case "DeleteMessage":
		Response, StatusCode = handlers.DeleteMessage(r.Form, &queues)
	case "DeleteMessageBatch":
		Response, StatusCode = handlers.DeleteMessageBatch(r.Form, &queues)
	case "DeleteQueue":
		Response, StatusCode = handlers.DeleteQueue(r.Form, &queues)
	case "GetQueueAttributes":
		Response, StatusCode = handlers.GetQueueAttributes(r.Form, &queues)
	case "GetQueueUrl":
		Response, StatusCode = handlers.GetQueueURL(r.Form, &queues)
	case "ListQueues":
		Response, StatusCode = handlers.ListQueues(r.Form, &queues)
	case "SendMessage":
		Response, StatusCode = handlers.SendMessage(r.Form, &queues)
	case "SendMessageBatch":
		Response, StatusCode = handlers.SendMessageBatch(r.Form, &queues)
	case "ReceiveMessage":
		Response, StatusCode = handlers.ReceiveMessage(r.Form, &queues)
	default:
		Response, StatusCode = util.Error("InvalidAction", "The action or operation requested is invalid. Verify that the action is typed correctly.")
	}

	w.WriteHeader(StatusCode)
	fmt.Fprintf(w, Response)
}

func main() {
	var Port = "8080"
	if len(os.Args) > 1 {
		Port = os.Args[1]
	}

	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":"+Port, nil))
}
