package protocol

import "encoding/json"

// Request is a JSON-RPC 2.0 request sent from kova to the extension.
// ID is omitted (null) for notifications, but all kova->extension messages
// are requests expecting a response.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      int64           `json:"id"`
}

// Response is a JSON-RPC 2.0 response sent from the extension to kova.
// Exactly one of Result or Error will be non-nil for a valid response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
	ID      int64           `json:"id"`
}

// Notification is a JSON-RPC 2.0 notification sent by the extension.
// Notifications have no ID and require no response.
type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface for RPCError.
func (e *RPCError) Error() string { return e.Message }
