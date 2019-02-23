package server

import "fmt"

// Response represents a response to a user request.
type Response struct {
	Req Request
}

// Success generates a successful response.
func (resp Response) Success(Action string, Result string) (Body string, Code int) {
	return fmt.Sprintf(`<%sResponse>
	<%sResult>%s</%sResult>
	<ResponseMetadata>
		<RequestId>%s</RequestId>
	</ResponseMetadata>
</%sResponse>`, Action, Action, Result, Action, resp.Req.ID, Action), 200
}

// Error generates an error response.
func (resp Response) Error(ErrorCode string, ErrorMessage string) (Body string, Code int) {
	return fmt.Sprintf(`<ErrorResponse>
  <Error>
    <Type>Sender</Type>
    <Code>%s</Code>
    <Message>%s</Message>
    <Detail/>
  </Error>
  <RequestId>%s</RequestId>
</ErrorResponse>`, ErrorCode, ErrorMessage, resp.Req.ID), 400
}
