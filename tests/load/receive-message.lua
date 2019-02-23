wrk.method = "POST"
wrk.body   = "Action=ReceiveMessage&QueueUrl=http://localhost/load_test_queue"
wrk.headers["Content-Type"] = "application/x-www-form-urlencoded"
