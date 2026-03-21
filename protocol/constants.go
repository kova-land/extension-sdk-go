package protocol

// Method names for kova -> extension requests (lifecycle).
const (
	MethodInitialize = "initialize"
	MethodShutdown   = "shutdown"
)

// Method names for kova -> extension requests (tools).
const (
	MethodToolCall = "tool_call"
)

// Method names for kova -> extension requests (HTTP routes).
const (
	MethodHTTPRequest = "http_request"
)

// Method names for kova -> extension requests (hooks).
const (
	MethodHookEvent = "hook_event"
)

// Method names for kova -> extension requests (services).
const (
	MethodServiceStart  = "service_start"
	MethodServiceHealth = "service_health"
	MethodServiceStop   = "service_stop"
)

// Method names for kova -> extension requests (channels).
const (
	MethodChannelSend           = "channel_send"
	MethodChannelTyping         = "channel_typing"
	MethodChannelReactionAdd    = "channel_reaction_add"
	MethodChannelReactionRemove = "channel_reaction_remove"
	MethodPromptUser            = "prompt_user"
	MethodChannelCapabilities   = "channel_capabilities"
)

// Method names for kova -> extension requests (providers).
const (
	// MethodProviderCreateMessage sends a message request and expects a complete response.
	MethodProviderCreateMessage = "provider_create_message"
	// MethodProviderStreamMessage sends a message request; the extension responds with
	// stream_chunk notifications followed by a stream_end notification, then the final
	// JSON-RPC result.
	MethodProviderStreamMessage = "provider_stream_message"
	// MethodProviderCapabilities queries the extension for its model capabilities.
	MethodProviderCapabilities = "provider_capabilities"
)

// Method names for kova -> extension requests (TTS).
const (
	MethodTTSSynthesize = "tts_synthesize"
	MethodTTSVoices     = "tts_voices"
)

// Notification methods sent by the extension (extension -> kova, no response expected).
const (
	NotificationLog            = "log"
	NotificationChannelMessage = "channel_message"
)

// Notification methods for provider streaming (extension -> kova).
const (
	// NotificationStreamChunk carries an incremental content chunk during streaming.
	NotificationStreamChunk = "stream_chunk"
	// NotificationStreamEnd signals that streaming is complete.
	NotificationStreamEnd = "stream_end"
	// NotificationStreamError signals a provider failure mid-stream.
	NotificationStreamError = "stream_error"
)

// ErrorCode values follow JSON-RPC 2.0 spec plus extension-specific codes.
const (
	// Standard JSON-RPC 2.0 error codes.
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603

	// Extension-specific error codes (server-defined range: -32000 to -32099).
	ErrCodeToolNotFound   = -32000
	ErrCodeToolFailed     = -32001
	ErrCodeNotReady       = -32002
	ErrCodeProviderFailed = -32010
	ErrCodeStreamFailed   = -32011
	ErrCodeModelNotFound  = -32012
	ErrCodeTTSFailed      = -32013
	ErrCodeVoiceNotFound  = -32014
	ErrCodeHookFailed     = -32020
	ErrCodeHTTPFailed     = -32021
	ErrCodeServiceFailed  = -32022
)
