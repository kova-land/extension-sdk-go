package protocol

// HookEventParams holds the payload for a "hook_event" request.
type HookEventParams struct {
	Event string         `json:"event"`
	Data  map[string]any `json:"data"`
}

// HookEventResult is the response payload for a "hook_event" request.
// Action controls kova's behavior after the hook runs.
type HookEventResult struct {
	// Action is one of "continue", "block". Empty string means "continue".
	Action  string `json:"action,omitempty"`
	Message string `json:"message,omitempty"`
}
