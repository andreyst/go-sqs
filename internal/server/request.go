package server

import (
	"net/url"
)

// Request represents a user request.
type Request struct {
	ID     string
	Params url.Values
}
