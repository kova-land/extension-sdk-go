package protocol

// ToolCallParams holds the payload for a "tool_call" request.
type ToolCallParams struct {
	Name    string         `json:"name"`
	Input   map[string]any `json:"input"`
	Context *CallContext   `json:"context,omitempty"`
}

// ToolCallResult is the response payload for a "tool_call" request.
type ToolCallResult struct {
	// Output is the text output from the tool, forwarded to the LLM.
	Output string `json:"output"`
	// IsError indicates the output describes an error condition.
	IsError bool `json:"is_error"`
}
