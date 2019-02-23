package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/andreyst/go-sqs/internal/server"
	uuid "github.com/satori/go.uuid"

	"github.com/andreyst/go-sqs/internal/handlers"
)

// Queues TODO: add comment
var queues sync.Map

func handler(w http.ResponseWriter, r *http.Request) {
	// TODO: Validate it is a POST request
	// TODO: Handle also GET parameters
	// TODO: Handle headers passed as GET parameters
	r.ParseForm()

	var req = server.Request{
		ID:     uuid.Must(uuid.NewV4()).String(),
		Params: r.Form,
	}
	var resp = server.Response{
		Req: req,
	}

	var Action = r.Form.Get("Action")
	var ResponseBody = ""
	var StatusCode = 0
	switch Action {
	case "CreateQueue":
		ResponseBody, StatusCode = handlers.CreateQueue(req, resp, &queues)
	case "DeleteMessage":
		ResponseBody, StatusCode = handlers.DeleteMessage(req, resp, &queues)
	case "DeleteMessageBatch":
		ResponseBody, StatusCode = handlers.DeleteMessageBatch(req, resp, &queues)
	case "DeleteQueue":
		ResponseBody, StatusCode = handlers.DeleteQueue(req, resp, &queues)
	case "GetQueueAttributes":
		ResponseBody, StatusCode = handlers.GetQueueAttributes(req, resp, &queues)
	case "GetQueueUrl":
		ResponseBody, StatusCode = handlers.GetQueueURL(req, resp, &queues)
	case "ListQueues":
		ResponseBody, StatusCode = handlers.ListQueues(req, resp, &queues)
	case "SendMessage":
		ResponseBody, StatusCode = handlers.SendMessage(req, resp, &queues)
	case "SendMessageBatch":
		ResponseBody, StatusCode = handlers.SendMessageBatch(req, resp, &queues)
	case "ReceiveMessage":
		ResponseBody, StatusCode = handlers.ReceiveMessage(req, resp, &queues)
	default:
		ResponseBody, StatusCode = resp.Error("InvalidAction", "The action or operation requested is invalid. Verify that the action is typed correctly.")
	}

	w.WriteHeader(StatusCode)
	fmt.Fprintf(w, ResponseBody)
}

func main() {
	var Port = "8080"
	if len(os.Args) > 1 {
		Port = os.Args[1]
	}

	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":"+Port, nil))
}
