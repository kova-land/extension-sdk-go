package protocol

// HTTPRequestParams holds the payload for an "http_request" request.
type HTTPRequestParams struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
	Body    []byte            `json:"body,omitempty"`
}

// HTTPResponseResult is the response payload for an "http_request" request.
type HTTPResponseResult struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       []byte            `json:"body,omitempty"`
}
