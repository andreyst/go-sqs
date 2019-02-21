-- A lua script for wrk to sned post requests
-- Use by passing as --script parameter to wrk
wrk.method = "POST"
wrk.body   = "Action=SendMessage&MessageBody=test1&QueueUrl=myqueue"
wrk.headers["Content-Type"] = "application/x-www-form-urlencoded"
