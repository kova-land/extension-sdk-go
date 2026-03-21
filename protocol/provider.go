package protocol

import "encoding/json"

// ProviderContentBlock is a single content block in a provider message.
// Mirrors provider.ContentBlock for bridge serialization.
type ProviderContentBlock struct {
	Type    string          `json:"type"`
	Text    string          `json:"text,omitempty"`
	ID      string          `json:"id,omitempty"`
	Name    string          `json:"name,omitempty"`
	Input   json.RawMessage `json:"input,omitempty"`
	Output  string          `json:"output,omitempty"`
	IsError bool            `json:"is_error,omitempty"`
}

// ProviderMessage is a single conversation turn sent over the bridge.
// Mirrors provider.Message for bridge serialization.
type ProviderMessage struct {
	Role    string                 `json:"role"`
	Content []ProviderContentBlock `json:"content"`
}

// ProviderToolDef describes a tool available to the LLM, sent over the bridge.
type ProviderToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// ProviderCreateMessageParams is the payload for a "provider_create_message" or
// "provider_stream_message" request.
type ProviderCreateMessageParams struct {
	// Model is the model identifier (e.g. "claude-sonnet-4-6").
	Model string `json:"model"`
	// MaxTokens is the maximum number of tokens to generate.
	MaxTokens int64 `json:"max_tokens"`
	// SystemPrompt is the system prompt text.
	SystemPrompt string `json:"system_prompt,omitempty"`
	// Messages is the conversation history.
	Messages []ProviderMessage `json:"messages"`
	// Tools is the set of tools available to the model.
	Tools []ProviderToolDef `json:"tools,omitempty"`
	// Context carries agent/session metadata.
	Context *CallContext `json:"context,omitempty"`
	// StreamID is set only for provider_stream_message requests. The extension
	// uses this ID in stream_chunk/stream_end/stream_error notifications so
	// the core can correlate chunks to the originating request.
	StreamID string `json:"stream_id,omitempty"`
}

// ProviderCreateMessageResult is the response payload for a
// "provider_create_message" request (and the final result of
// "provider_stream_message" after all stream notifications).
type ProviderCreateMessageResult struct {
	// Content is the model's response content blocks.
	Content []ProviderContentBlock `json:"content"`
	// StopReason indicates why the model stopped (e.g. "end_turn", "tool_use", "max_tokens").
	StopReason string `json:"stop_reason"`
	// InputTokens is the number of input tokens consumed.
	InputTokens int64 `json:"input_tokens,omitempty"`
	// OutputTokens is the number of output tokens generated.
	OutputTokens int64 `json:"output_tokens,omitempty"`
}

// StreamChunkParams is the payload for a "stream_chunk" notification.
type StreamChunkParams struct {
	// StreamID correlates this chunk to the originating provider_stream_message request.
	StreamID string `json:"stream_id"`
	// Type is the event type (e.g. "text_delta", "tool_use_start").
	Type string `json:"type"`
	// Text carries the incremental text for text_delta events.
	Text string `json:"text,omitempty"`
	// ToolID is set for tool_use_start events.
	ToolID string `json:"tool_id,omitempty"`
	// ToolName is set for tool_use_start events.
	ToolName string `json:"tool_name,omitempty"`
	// ToolInput carries incremental JSON for tool input deltas.
	ToolInput string `json:"tool_input,omitempty"`
	// Index is the zero-based sequence number of this chunk within the stream.
	Index int `json:"index"`
}

// StreamEndParams is the payload for a "stream_end" notification.
type StreamEndParams struct {
	// StreamID correlates this notification to the originating request.
	StreamID string `json:"stream_id"`
	// TotalChunks is the number of stream_chunk notifications sent.
	TotalChunks int `json:"total_chunks"`
}

// StreamErrorParams is the payload for a "stream_error" notification.
type StreamErrorParams struct {
	// StreamID correlates this notification to the originating request.
	StreamID string `json:"stream_id"`
	// Code is an error code (uses standard JSON-RPC or extension-specific codes).
	Code int `json:"code"`
	// Message describes the error.
	Message string `json:"message"`
}

// ProviderCapabilitiesParams is the payload for a "provider_capabilities" request.
type ProviderCapabilitiesParams struct {
	// Model is the model to query capabilities for. If empty, the extension
	// returns default capabilities.
	Model string `json:"model,omitempty"`
}

// ProviderCapabilitiesResult is the response payload for a "provider_capabilities" request.
type ProviderCapabilitiesResult struct {
	// Models lists the model identifiers this provider supports.
	Models []string `json:"models,omitempty"`
	// SupportsStreaming indicates whether the provider supports streaming responses.
	SupportsStreaming bool `json:"supports_streaming"`
	// SupportsVision indicates whether the provider accepts image inputs.
	SupportsVision bool `json:"supports_vision,omitempty"`
	// SupportsAudioOutput indicates whether the provider can generate audio.
	SupportsAudioOutput bool `json:"supports_audio_output,omitempty"`
	// MaxImages is the maximum number of images per request (0 = no limit).
	MaxImages int `json:"max_images,omitempty"`
	// MaxTokens is the model's maximum output token limit.
	MaxTokens int64 `json:"max_tokens,omitempty"`
	// ContextWindow is the model's total context window size in tokens.
	ContextWindow int64 `json:"context_window,omitempty"`
}
