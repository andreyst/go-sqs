# go-sqs
An in-memory implementation of Amazon SQS. Created for testing load tester and educational purposes.

## Running
Execute `go run sqs-go/main.go`. Go-sqs launches on port 8080 by default.

## Developing
For development install [modd](https://github.com/cortesi/modd) and run tests and program with `modd notify`.

## Compatibility
In short, a lot of stuff is not implemented, most notably all batch methods, tag/untag, permissions methods, authentication, and FIFO queues.
