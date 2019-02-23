wrk.method = "POST"
wrk.body   = "Action=SendMessage&MessageBody=test1&QueueUrl=http://localhost/load_test_queue"
wrk.headers["Content-Type"] = "application/x-www-form-urlencoded"
